package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
)

const currency = "RUB"

type bnResp struct {
	Price float64 `json:"price,string"`
	Code  int64   `json:"code"`
}

type wallet map[string]float64

var db = map[int64]wallet{}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {

	bot, err := tgbotapi.NewBotAPI(getBotToken())
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		command := strings.Split(update.Message.Text, " ")

		switch command[0] {
		case "ADD":
			if len(command) != 3 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Incorrect command!"))
			}
			amount, err := strconv.ParseFloat(command[2], 64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Incorrect command format!"))
			}
			if _, ok := db[update.Message.Chat.ID]; !ok {
				db[update.Message.Chat.ID] = wallet{}
			}

			db[update.Message.Chat.ID][command[1]] += amount
			balanceText := fmt.Sprintf("%s: balance is: %.4f", command[1], db[update.Message.Chat.ID][command[1]])
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Currency added! "+balanceText))
		case "SUB":
			if len(command) != 3 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Incorrect command!"))
			}
			amount, err := strconv.ParseFloat(command[2], 64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Incorrect command format!"))
			}
			if _, ok := db[update.Message.Chat.ID]; !ok {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Wallet doesn't contain symbol: %s!", command[1])))
			}

			db[update.Message.Chat.ID][command[1]] -= amount
			balanceText := fmt.Sprintf("%s: balance is: %.4f", command[1], db[update.Message.Chat.ID][command[1]])
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Balance changed! "+balanceText))
		case "DEL":
			if len(command) != 2 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Incorrect command!"))
			}

			delete(db[update.Message.Chat.ID], command[1])
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Deleted: "+command[1]))
		case "SHOW":
			msg := ""
			var sumTotal float64

			for key, amount := range db[update.Message.Chat.ID] {
				price, _ := getPrice(key, currency)
				sumTotal += amount * price
				msg += fmt.Sprintf("%s: %.4f [%.2f %s]\n", key, amount, amount*price, currency)
			}
			msg += fmt.Sprintf("Total: %.2f %s", sumTotal, currency)
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		default:
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Команда не найдена!"))
		}

	}

}

func getPrice(symbol string, currency string) (price float64, err error) {

	var currencySuffix string
	if currency == "USD" {
		currencySuffix = "T"
	}

	priceUrl := fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s%s%s", symbol, currency, currencySuffix)
	resp, err := http.Get(priceUrl)
	if err != nil {
		log.Print(err)
		return
	}

	defer resp.Body.Close()

	var jsonResp bnResp
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		log.Print(err)
		return
	}

	if jsonResp.Code != 0 {
		err = errors.New(fmt.Sprintf("Symbol is incorrect: %s", symbol))
		log.Print(err)
	}

	price = jsonResp.Price
	return
}

func getBotToken() string {
	token, exists := os.LookupEnv("BOT_TOKEN")

	if !exists {
		log.Panic("Not found environment variable: BOT_TOKEN")
	}

	return token
}
