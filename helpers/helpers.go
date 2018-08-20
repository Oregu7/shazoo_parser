package helpers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	tgcl "github.com/meinside/telegraph-go/client"
	"github.com/mmcdole/gofeed"
	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/telegram-bot-api.v4"
)

// Article статья
type Article struct {
	title, image, text string
	tags               []string
}

// ScrapArticle парсит статью
func ScrapArticle(url string) Article {
	// Request the HTML page.
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	text := ""
	title := strings.TrimSpace(doc.Find("div.entryContextHeader > h1").Text())
	image, _ := doc.Find("div.entryHeaderContainer.entryImageContainer > img").Attr("src")
	tags := doc.Find("section.tags > ul.inline > li > a").Map(func(_ int, s *goquery.Selection) string {
		return "#" + s.Text()
	})
	doc.Find("section.body").Children().Each(func(_ int, s *goquery.Selection) {
		storyImage, existImage := s.Find("img").Attr("src")
		if existImage {
			text += fmt.Sprintf("<figure><img src='%s'></figure>", storyImage)
		} else if s.HasClass("twitter-tweet") {
			twitter, _ := s.Find("a").Last().Attr("href")
			text += fmt.Sprintf("<figure><iframe src='/embed/twitter?url=%s'></iframe></figure>", twitter)
		} else if s.HasClass("videoPlayer") {
			regx := regexp.MustCompile(`https://www.youtube.com/embed/(\w+)\?.*`)
			youtube, _ := s.Find("iframe").Attr("src")
			match := regx.FindStringSubmatch(youtube)
			shortURL := "https://www.youtube.com/watch?v=" + match[1]
			text += fmt.Sprintf("<figure><iframe src='/embed/youtube?url=%s'></iframe></figure>", shortURL)
		} else if goquery.NodeName(s) == "ul" {
			list := s.Find("li").Map(func(_ int, s *goquery.Selection) string {
				return "<li>" + strings.TrimSpace(s.Text()) + "</li>"
			})
			text += "<ul>" + strings.Join(list, "") + "</ul>"
		} else {
			nodeName := goquery.NodeName(s)
			ret, _ := s.Html()
			text += fmt.Sprintf("<%s>%s</%s>", nodeName, strings.TrimSpace(ret), nodeName)
		}
	})

	return Article{title, image, text, tags}
}

// PostToTelegraph постит статью в telegra.ph
func PostToTelegraph(articel Article) string {
	client, _ := tgcl.Load(os.Getenv("TELEGRAPH_TOKEN"))
	html := fmt.Sprintf("<figure><img src='%s'></figure><div>%s</div>", articel.image, articel.text)
	page, _ := client.CreatePageWithHtml(articel.title, "Shazoo", "https://t.me/shazoo_news", html, true)

	return page.Url
}

// GetMD5Hash возвращает хеш строки
func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

// GetUpdates возвращает новые статьи
func GetUpdates(db *leveldb.DB, items []*gofeed.Item) []string {
	updates := make([]string, 0)
	newPatternStr := ""
	pattern, _ := db.Get([]byte("shazoo"), nil)
	for i, item := range items {
		// ищем обновления
		linkHash := GetMD5Hash(item.Link)
		mtch, _ := regexp.MatchString(string(pattern), linkHash)
		if len(pattern) == 0 || !mtch {
			updates = append(updates, item.Link)
		}
		// формируем новый паттерн с текущими данными
		newPatternStr += linkHash
		if i < len(items)-1 {
			newPatternStr += "|"
		}
	}
	// сохраняем новый паттерн
	db.Put([]byte("shazoo"), []byte(newPatternStr), nil)
	return updates
}

// PostToChannel постим новость в канал
func PostToChannel(bot *tgbotapi.BotAPI, url string) {
	article := ScrapArticle(url)
	telegraphURL := PostToTelegraph(article)
	// post to channel
	text := fmt.Sprintf("<a href='%s'>%s</a>", telegraphURL, article.title)
	msg := tgbotapi.NewMessageToChannel(os.Getenv("CHANNEL"), text)
	msg.ParseMode = "HTML"
	bot.Send(msg)
}
