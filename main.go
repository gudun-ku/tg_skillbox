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
		chatId := update.Message.Chat.ID

		switch command[0] {
		case "ADD":
			answer := addSymbol(command, update.Message.Chat.ID)
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, answer))
		case "SUB":
			answer := subSymbol(command, update.Message.Chat.ID)
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, answer))
		case "DEL":
			answer := deleteSymbol(command, chatId)
			bot.Send(tgbotapi.NewMessage(chatId, answer))
		case "SHOW":
			answer := showWallet(chatId)
			bot.Send(tgbotapi.NewMessage(chatId, answer))
		default:
			bot.Send(tgbotapi.NewMessage(chatId, "Команда не найдена!"))
		}

	}

}

func addSymbol(command []string, chatId int64) (answer string) {

	if len(command) != 3 {
		return fmt.Sprintf("Incorrect command: %v", command)
	}
	amount, err := strconv.ParseFloat(command[2], 64)
	if err != nil {
		return fmt.Sprintf("Incorrect command format: %v", command[2])
	}
	if _, ok := db[chatId]; !ok {
		db[chatId] = wallet{}
	}

	db[chatId][command[1]] += amount
	balanceText := fmt.Sprintf("%s: balance is: %.4f", command[1], db[chatId][command[1]])
	return "Currency added! " + balanceText
}

func subSymbol(command []string, chatId int64) (answer string) {

	if len(command) != 3 {
		return fmt.Sprintf("Incorrect command: %v", command)
	}
	amount, err := strconv.ParseFloat(command[2], 64)
	if err != nil {
		return fmt.Sprintf("Incorrect command format: %v", command[2])
	}
	if _, ok := db[chatId]; !ok {
		return fmt.Sprintf("Wallet doesn't contain symbol: %s!", command[1])
	}

	curAmount := db[chatId][command[1]]
	if curAmount > amount {
		db[chatId][command[1]] -= amount
		balanceText := fmt.Sprintf("%s: balance is: %.4f", command[1], db[chatId][command[1]])
		return fmt.Sprintf("Change fixed: %s", balanceText)
	} else if curAmount == amount {
		delete(db[chatId], command[1])
		return "Deleted: " + command[1]
	} else {
		return fmt.Sprintf("Not enough amout on account: %s", command[1])
	}
}

func deleteSymbol(command []string, chatId int64) (answer string) {

	if len(command) != 2 {
		return fmt.Sprintf("Incorrect command: %v", command)
	}

	delete(db[chatId], command[1])
	return fmt.Sprintf("Deleted: %s", command[1])
}

func showWallet(chatId int64) (answer string) {

	msg := ""
	var sumTotal float64

	for key, amount := range db[chatId] {
		price, _ := getPrice(key, currency)
		sumTotal += amount * price
		msg += fmt.Sprintf("%s: %.4f [%.2f %s]\n", key, amount, amount*price, currency)
	}
	msg += fmt.Sprintf("Total: %.2f %s", sumTotal, currency)

	return msg
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
