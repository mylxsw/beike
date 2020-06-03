package main

import (
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/gocolly/colly"
	"github.com/mylxsw/asteria/log"
	"strconv"
	"strings"
	"time"
)

var COLUMNS = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func main() {

	// 成交小区查询 https://m.ke.com/cq/sold/search/

	// c367104056275661or1 绿云绣
	// c3611063874373or1 春风与湖
	// c3611063794230or1 北大资源燕南
	// c3611064001735or1 渝高香洲
	// c3611060917581or1 同天观云邸
	// c3611063899554or1 渝高幸福九里

	// 小区id
	areaID := "c3611063899554or1"
	// 小区名称
	areaName := "渝高幸福九里"

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
		detailURL := fmt.Sprintf("https://m.ke.com/cq/chengjiao/%s.html", element.Attr("data-id"))
		err := detailCollector.Visit(detailURL)
		if err != nil {
			log.Errorf("visit detail page %s, error: %v", detailURL, err)
		}
	})

	saveFile := excelize.NewFile()
	saveFile.NewSheet("Sheet1")

	saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[0]), 1), "名称")
	saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[1]), 1), "成交价")
	saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[2]), 1), "挂牌价")
	saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[3]), 1), "单价")
	saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[4]), 1), "成交日期")
	saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[5]), 1), "成交周期")
	saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[6]), 1), "调价次数")
	saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[7]), 1), "最终差价")

	rowIndex := 2
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

		saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[0]), rowIndex), houseName)
		saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[1]), rowIndex), dealPrice)
		saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[2]), rowIndex), stickerPrice)
		saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[3]), rowIndex), unitPrice)
		saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[4]), rowIndex), dealDate)
		saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[5]), rowIndex), dealDays)
		saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[6]), rowIndex), changePriceTimes)
		saveFile.SetCellValue("Sheet1", fmt.Sprintf("%s%d", string(COLUMNS[7]), rowIndex), stickerPriceFloat-dealPriceFloat)

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

		rowIndex++
	})

	if err := c.Visit(fmt.Sprintf("https://m.ke.com/cq/chengjiao/%s/", areaID)); err != nil {
		panic(err)
	}

	if err := saveFile.SaveAs(areaName + ".xlsx"); err != nil {
		panic(err)
	}

}
