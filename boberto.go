package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
	"gopkg.in/gomail.v2"
)

type Match struct {
	Hour    string
	Home    string
	Out     string
	HomeOdd string
	DrawOdd string
	OutOdd  string
}

func main() {
	godotenv.Load()
	c := colly.NewCollector(colly.AllowedDomains(os.Getenv("firstDomain"), os.Getenv("secondDomain")))
	matches := getMatches(c)
	csv := generateCsv(matches)
	sendMatchesInEmail(csv)
}

func getDayMatchUrl(c *colly.Collector) string {
	var dayMatchLink string

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting ", r.URL)
	})

	c.OnHTML("div.leaguesN-item .leaguesN-item-body .cardItem", func(e *colly.HTMLElement) {
		if e.Index == 0 {
			dayMatchLink = substr(e.Attr("onclick"), 17, 61)
		}
	})

	c.Visit(os.Getenv("firstUrl"))

	return dayMatchLink
}

func getMatches(c *colly.Collector) []Match {
	dayMatchesUrl := getDayMatchUrl(c)
	var matches []Match

	c.OnRequest(func(r *colly.Request) {
		fmt.Printf("Visit %s", r.URL)
	})

	c.OnHTML(".eventlistContainer", func(e *colly.HTMLElement) {
		e.ForEach(".containerCards #cardJogo", func(i int, ee *colly.HTMLElement) {
			var match Match
			ee.ForEach(".dateAndHour", func(i int, eee *colly.HTMLElement) {
				match.Hour = eee.ChildText(".hour")
			})

			ee.ForEach(".teams", func(i int, eee *colly.HTMLElement) {
				eee.ForEach(".team", func(i int, eeee *colly.HTMLElement) {
					if i == 0 {
						match.Home = eeee.Text
					} else {
						match.Out = eeee.Text
					}
				})
			})

			ee.ForEach(".outcomesMain", func(i int, eee *colly.HTMLElement) {
				eee.ForEach(".odd", func(i int, eeee *colly.HTMLElement) {
					if i == 0 {
						match.HomeOdd = eeee.Text
					}
					if i == 1 {
						match.DrawOdd = eeee.Text
					}
					if i == 2 {
						match.OutOdd = eeee.Text
					}
				})
			})

			matches = append(matches, match)
		})
	})

	c.Visit(os.Getenv("baseUrl") + dayMatchesUrl)

	return matches
}

func generateCsv(matches []Match) []byte {
	// file, err := os.Create("matches.csv")
	// if err != nil {
	// 	log.Fatal("Erro durante a criação do arquivo CSV")
	// }
	// defer file.Close()
	var csvBuffer bytes.Buffer

	writer := csv.NewWriter(&csvBuffer)
	headers := []string{
		"Horário",
		"Casa",
		"Visitante",
		"Odd Casa",
		"Odd Empate",
		"Odd Visitante",
	}
	writer.Write(headers)

	for _, match := range matches {
		record := []string{
			match.Hour,
			match.Home,
			match.Out,
			strings.Replace(match.HomeOdd, ",", ".", -1),
			strings.Replace(match.DrawOdd, ",", ".", -1),
			strings.Replace(match.OutOdd, ",", ".", -1),
		}

		writer.Write(record)
	}

	writer.Flush()

	return csvBuffer.Bytes()
}

func sendMatchesInEmail(csv []byte) {
	msg := gomail.NewMessage()
	msg.SetHeader("From", os.Getenv("emailFrom"))
	msg.SetHeader("To", os.Getenv("emailTo"))
	msg.SetHeader("Subject", "Jogos de Hoje - Boberto")
	msg.SetBody("text/html", fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="pt-br">
		<head>
				<meta charset="UTF-8">
				<meta http-equiv="X-UA-Compatible" content="IE=edge">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<title>Saudações e Boas Apostas!</title>
		</head>
		<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; color: #333; padding: 20px;">

				<div style="max-width: 600px; margin: 0 auto; background-color: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 0 10px rgba(0,0,0,0.1);">
						<h1 style="font-size: 24px; margin-bottom: 20px;">Olá, %s!</h1>

						<p style="font-size: 16px;">Espero que este e-mail o encontre bem.</p>

						<p style="font-size: 16px;">Segue em anexo o arquivo com as partidas de futebol de hoje.</p>

						<p style="font-size: 16px;">Desejamos a você boas apostas!</p>

						<p style="font-size: 16px;">Atenciosamente,<br> %s</p>
				</div>

		</body>
		</html>
	`, "Marcelo", "Boberto"))

	msg.Attach("partidas.csv", gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := w.Write(csv)
		return err
	}))

	dialer := gomail.NewDialer("smtp.gmail.com", 587, os.Getenv("emailUsername"), os.Getenv("emailPassword"))

	if err := dialer.DialAndSend(msg); err != nil {
		log.Fatal("Erro durante o envio do email.")
	}

	fmt.Println("\n Email enviado com sucesso!")
}

func substr(str string, start, end int) string {
	return strings.TrimSpace(str[start:end])
}
