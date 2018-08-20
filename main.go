package main

import (
	"fmt"
	"log"
	"shazoo/helpers"

	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/telegram-bot-api.v4"
)

func main() {
	godotenv.Load()
	fp := gofeed.NewParser()
	db, err := leveldb.OpenFile("storage.db", nil)
	if err != nil {
		log.Panic(err)
	}
	/*bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	gocron.Every(20).Minutes().Do(listener, fp, db, bot)
	<-gocron.Start()*/
	test(fp, db)
}

func listener(fp *gofeed.Parser, db *leveldb.DB, bot *tgbotapi.BotAPI) {
	feed, _ := fp.ParseURL("https://shazoo.ru/feed/rss")
	updates := helpers.GetUpdates(db, feed.Items)
	fmt.Printf("updates => %v\n", len(updates))
	for _, url := range updates {
		helpers.PostToChannel(bot, url)
	}
}

func test(fp *gofeed.Parser, db *leveldb.DB) {
	godotenv.Load()
	//feed, _ := fp.ParseURL("https://shazoo.ru/feed/rss")
	article := helpers.ScrapArticle("https://shazoo.ru/2018/08/20/69220/novyj-trejler-i-skrinshoty-world-war-z")
	//https://shazoo.ru/2018/08/20/69218/novyj-gejmplejnyj-trejler-world-war-3
	//https://shazoo.ru/2018/08/20/69227/otkrytaya-beta-battlefield-5-startuet-v-sentyabre-demonstraciya-trassirovki-luchej
	url := helpers.PostToTelegraph(article)
	fmt.Println(url)
}
