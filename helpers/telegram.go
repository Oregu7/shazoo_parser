package helpers

import (
	"fmt"
	"os"
	"strings"

	tgcl "github.com/meinside/telegraph-go/client"
	"gopkg.in/telegram-bot-api.v4"
)

// PostToTelegraph постит статью в telegra.ph
func PostToTelegraph(articel Article) string {
	client, _ := tgcl.Load(os.Getenv("TELEGRAPH_TOKEN"))
	html := fmt.Sprintf("<figure><img src='%s'></figure><div>%s</div>", articel.image, articel.text)
	page, _ := client.CreatePageWithHtml(articel.title, "Shazoo", "https://t.me/shazoo_news", html, true)

	return page.Url
}

// PostToChannel постим новость в канал
func PostToChannel(bot *tgbotapi.BotAPI, url string) {
	article := ScrapArticle(url)
	telegraphURL := PostToTelegraph(article)
	// post to channel
	tags := strings.Join(article.tags, " ")
	text := fmt.Sprintf("<a href='%s'>%s</a>\n\n%s", telegraphURL, article.title, tags)
	msg := tgbotapi.NewMessageToChannel(os.Getenv("CHANNEL"), text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup([]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonURL("Читать на сайте", url)})
	bot.Send(msg)
}
