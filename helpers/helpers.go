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
	// tags
	regxTagReplace := regexp.MustCompile(`[\s-:']+`)
	regxTag := regexp.MustCompile(`.+\/([\w-]+)`)
	tags := doc.Find("section.tags > ul.inline > li > a").Map(func(_ int, s *goquery.Selection) string {
		text, _ := s.Attr("href")
		match := regxTag.FindStringSubmatch(text)
		return "#" + regxTagReplace.ReplaceAllString(match[1], "_")
	})
	// body
	doc.Find("section.body").Children().Each(func(_ int, s *goquery.Selection) {
		storyImage, existImage := s.Find("img").Attr("src")
		if s.HasClass("reference") {
			text += ""
		} else if s.HasClass("gallery") {
			images := s.Find(".galleryItems > li > a").Map(func(_ int, s *goquery.Selection) string {
				mediaImage, _ := s.Attr("href")
				return fmt.Sprintf("<figure><img src='%s'></figure>", mediaImage)
			})

			text += strings.Join(images, "")
		} else if existImage {
			text += fmt.Sprintf("<figure><img src='%s'></figure>", storyImage)
		} else if s.HasClass("twitter-tweet") {
			twitter, _ := s.Find("a").Last().Attr("href")
			text += fmt.Sprintf("<figure><iframe src='/embed/twitter?url=%s'></iframe></figure>", twitter)
		} else if s.HasClass("videoPlayer") {
			videoURL, _ := s.Find("iframe").Attr("src")
			text += createVideoFrame(videoURL)
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

	doc.Find("section.media").Children().Each(func(_ int, s *goquery.Selection) {
		if s.HasClass("gallery") {
			images := s.Find(".galleryItems > li > a").Map(func(_ int, s *goquery.Selection) string {
				mediaImage, _ := s.Attr("href")
				return fmt.Sprintf("<figure><img src='%s'></figure>", mediaImage)
			})

			text += strings.Join(images, "")
		} else if s.HasClass("videoPlayer") {
			videoURL, _ := s.Find("iframe").Attr("src")
			text += createVideoFrame(videoURL)
		}
	})

	return Article{title, image, text, tags}
}

// PostToTelegraph постит статью в telegra.ph
func PostToTelegraph(articel Article) string {
	fmt.Println(articel.tags)
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
	tags := strings.Join(article.tags, " ")
	text := fmt.Sprintf("%s\n\n<a href='%s'>%s</a>", tags, telegraphURL, article.title)
	msg := tgbotapi.NewMessageToChannel(os.Getenv("CHANNEL"), text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup([]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonURL("Читать на сайте", url)})
	bot.Send(msg)
}

func createVideoFrame(url string) string {
	youtubeRegx := regexp.MustCompile(`https://www.youtube.com/embed/([-\w]+)\?.*`)
	serviceName := "youtube"
	shortURL := url

	isTwich, _ := regexp.MatchString("twitch", url)
	if isTwich {
		serviceName = "twitch"
		return fmt.Sprintf("<a href = '%s'>%s</a>", url, url)
	}

	match := youtubeRegx.FindStringSubmatch(url)
	shortURL = "https://www.youtube.com/watch?v=" + match[1]

	return fmt.Sprintf("<figure><iframe src='/embed/%s?url=%s'></iframe></figure>", serviceName, shortURL)
}
