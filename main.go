package main

import (
	"database/sql"
	"fmt"
	"github.com/mylxsw/beike/models"
	"github.com/mylxsw/eloquent/migrate"
	"github.com/mylxsw/eloquent/query"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gocolly/colly"
	"github.com/mylxsw/asteria/log"
)

var COLUMNS = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func main() {

	// 成交小区查询 https://m.ke.com/cq/sold/search/

	areas := map[string]string{
		"c367104056275661or1": "绿云绣",
		"c3611063874373or1":   "春风与湖",
		"c3611063794230or1":   "北大资源燕南",
		"c3611064001735or1":   "渝高香洲",
		"c3611060917581or1":   "同天观云邸",
		"c3611063899554or1":   "渝高幸福九里",
		"c3611063770554or1":   "东原九城时光",
		"c3611056573837or1":   "华宇北城中央",
		"c3611063505205or1":   "奥园盘龙壹号",
		"c3611059984109or1":   "蜜城",
		"c3611060811559or1":   "盛美居",
	}

	connURI := "mylxsw:@tcp(127.0.0.1:3306)/beike?parseTime=true"
	db, err := sql.Open("mysql", connURI)
	if err != nil {
		panic(err)
	}

	defer db.Close()
	createMigrate(db)

	for areaID, areaName := range areas {
		handleArea(db, areaID, areaName)
	}
}

func createMigrate(db *sql.DB) {
	m := migrate.NewManager(db).Init()
	m.Schema("202004062122").Create("bk_deal_history", func(builder *migrate.Builder) {
		builder.Increments("id")
		builder.String("area_id", 255)
		builder.String("area_name", 255)
		builder.String("house_id", 255).Unique()
		builder.String("name", 255)
		builder.Float("deal_price", 0, 2)
		builder.Float("sticker_price", 0, 2)
		builder.Float("unit_price", 0, 2).Nullable(true)
		builder.Date("deal_date").Nullable(true)
		builder.Integer("deal_days", false, true).Nullable(true)
		builder.Integer("change_price_times", false, true).Nullable(true)
		builder.Timestamps(0)
	})
	m.Schema("20200406212202").Table("bk_deal_history", func(builder *migrate.Builder) {
		builder.String("house_type", 255)
		builder.Float("house_size", 0, 2)
	})

	if err := m.Run(); err != nil {
		panic(err)
	}
}

func handleArea(db *sql.DB, areaID string, areaName string) {
	dealHistoryModel := models.NewDealHistoryModel(db)

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36"),
		colly.MaxDepth(2),
	)

	if err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1,
		RandomDelay: 5 * time.Second,
	}); err != nil {
		panic(err)
	}

	detailCollector := c.Clone()

	c.OnHTML("div.kem__chengjiao-house-tile", func(element *colly.HTMLElement) {
		houseID := element.Attr("data-id")
		exist, err := dealHistoryModel.Exists(query.Builder().Where("house_id", houseID))
		if err != nil {
			panic(err)
		}

		if exist {
			return
		}

		detailURL := fmt.Sprintf("https://m.ke.com/cq/chengjiao/%s.html", houseID)
		if err := detailCollector.Visit(detailURL); err != nil {
			log.Errorf("visit detail page %s, error: %v", detailURL, err)
		}
	})
	detailCollector.OnHTML(".content_area", func(element *colly.HTMLElement) {
		houseName := element.ChildText(".house_head .house_desc")
		dealPrice := element.ChildText(".house_head .similar_data .similar_data_detail:nth-child(1) span[data-mark=price]")
		unitPrice := strings.TrimSuffix(element.ChildText(".house_head .similar_data .similar_data_detail:nth-child(2) p:nth-child(2)"), "元/平")

		dealDate := element.ChildText(".house_head .house_description li:nth-child(2)")
		dealDate = strings.TrimPrefix(dealDate, "成交：")

		stickerPrice := element.ChildText(".deal_mod .mod_cont .data:nth-child(1) .box_col:nth-child(1) strong")
		dealDays := element.ChildText(".deal_mod .mod_cont .data:nth-child(1) .box_col:nth-child(2) strong")
		changePriceTimes := element.ChildText(".deal_mod .mod_cont .data:nth-child(1) .box_col:nth-child(3) strong")

		dealPriceFloat, _ := strconv.ParseFloat(dealPrice, 32)
		stickerPriceFloat, _ := strconv.ParseFloat(stickerPrice, 32)
		unitPriceFloat, _ := strconv.ParseFloat(unitPrice, 32)
		dealDaysInt, _ := strconv.Atoi(dealDays)
		changePriceTimesInt, _ := strconv.Atoi(changePriceTimes)

		names := strings.Split(houseName, " ")
		houseType := names[1]
		houseSize, _ := strconv.ParseFloat(strings.TrimSuffix(names[2], "m²"), 32)

		dealDateD, _ := time.Parse("2006-01-02", strings.ReplaceAll(dealDate, ".", "-"))

		_, err := dealHistoryModel.Save(models.DealHistory{
			AreaId:           areaID,
			AreaName:         areaName,
			HouseId:          strings.TrimSuffix(strings.Split(element.Request.URL.Path, "/")[3], ".html"),
			Name:             houseName,
			DealPrice:        dealPriceFloat,
			StickerPrice:     stickerPriceFloat,
			UnitPrice:        unitPriceFloat,
			DealDate:         dealDateD,
			DealDays:         int64(dealDaysInt),
			ChangePriceTimes: int64(changePriceTimesInt),
			HouseSize:        houseSize,
			HouseType:        houseType,
		})
		if err != nil {
			log.Errorf(
				"差价：%.2f, 名称：%s, 成交价：%s, 挂牌价：%s, 单价：%s, 成交日期：%s, 成交周期：%s, 调价次数：%s， ERROR: %v",
				stickerPriceFloat-dealPriceFloat,
				houseName,
				dealPrice,
				stickerPrice,
				unitPrice,
				dealDate,
				dealDays,
				changePriceTimes,
				err,
			)
		} else {
			log.Debugf(
				"差价：%.2f, 名称：%s, 成交价：%s, 挂牌价：%s, 单价：%s, 成交日期：%s, 成交周期：%s, 调价次数：%s",
				stickerPriceFloat-dealPriceFloat,
				houseName,
				dealPrice,
				stickerPrice,
				unitPrice,
				dealDate,
				dealDays,
				changePriceTimes,
			)
		}
	})

	if err := c.Visit(fmt.Sprintf("https://m.ke.com/cq/chengjiao/%s/", areaID)); err != nil {
		panic(err)
	}
}
