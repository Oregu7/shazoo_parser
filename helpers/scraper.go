package helpers

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func getDocument(url string) *goquery.Document {
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

	return doc
}

// ScrapArticle парсит статью
func ScrapArticle(url string) Article {
	// Request the HTML page.
	doc := getDocument(url)

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

func createVideoFrame(url string) string {
	youtubeRegx := regexp.MustCompile(`https://www.youtube.com/embed/([-\w]+)\?.*`)

	isYoutube, _ := regexp.MatchString("youtube", url)
	if isYoutube {
		match := youtubeRegx.FindStringSubmatch(url)
		shortURL := "https://www.youtube.com/watch?v=" + match[1]

		return fmt.Sprintf("<figure><iframe src='/embed/youtube?url=%s'></iframe></figure>", shortURL)
	}

	return fmt.Sprintf("<a href = '%s'>%s</a>", url, url)
}
