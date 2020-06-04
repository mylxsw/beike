
run:
	go run main.go

build-orm:
	orm models/*.yml

dump-sql:
	 mysqldump -c  --compact --extended-insert=false -u mylxsw beike > beike.sql