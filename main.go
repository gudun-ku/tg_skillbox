package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"github.com/pplcc/plotext/custplotter"
	"gonum.org/v1/plot"
)

const currency = "RUB"
const picFileName = "candlesticks.png"

type bnResp struct {
	Price float64 `json:"price,string"`
	Code  int64   `json:"code"`
}

// Kline define kline info
type Kline struct {
	OpenTime                 int64  `json:"openTime"`
	Open                     string `json:"open"`
	High                     string `json:"high"`
	Low                      string `json:"low"`
	Close                    string `json:"close"`
	Volume                   string `json:"volume"`
	CloseTime                int64  `json:"closeTime"`
	QuoteAssetVolume         string `json:"quoteAssetVolume"`
	TradeNum                 int64  `json:"tradeNum"`
	TakerBuyBaseAssetVolume  string `json:"takerBuyBaseAssetVolume"`
	TakerBuyQuoteAssetVolume string `json:"takerBuyQuoteAssetVolume"`
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
		case "GRAPH":
			err := getGraph(command, currency, chatId, bot)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(chatId, "Error when trying to send candles Graph"))
			}

		default:
			bot.Send(tgbotapi.NewMessage(chatId, "No such command!"))
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

func newJSON(data []byte) (j *simplejson.Json, err error) {
	j, err = simplejson.NewJson(data)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func getLowestTime() int64 {
	return (time.Now().Add(-24 * time.Hour)).UnixNano() / int64(time.Millisecond)
}

func getGraph(command []string, currency string, chatId int64, bot *tgbotapi.BotAPI) (err error) {

	if len(command) != 2 {
		return fmt.Errorf("Incorrect command: %v", command)
	}

	symbol := command[1]

	var currencySuffix string
	if currency == "USD" {
		currencySuffix = "T"
	}

	apiUrl := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=%s%s%s&interval=30m&startTime=%d", symbol, currency, currencySuffix, getLowestTime())
	resp, err := http.Get(apiUrl)
	if err != nil {
		log.Print(err)
		return err
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return err
	}

	j, err := newJSON(data)
	if err != nil {
		log.Print(err)
		return err
	}

	num := len(j.MustArray())

	res := make(custplotter.TOHLCVs, num)
	for i := 0; i < num; i++ {
		item := j.GetIndex(i)
		if len(item.MustArray()) < 11 {
			return fmt.Errorf("invalid kline response")
		}
		// Time
		res[i].T = float64(item.GetIndex(0).MustInt64())
		// Open
		res[i].O, err = strconv.ParseFloat(item.GetIndex(1).MustString(), 64)
		if err != nil {
			return fmt.Errorf("invalid kline response")
		}
		// Highest
		res[i].H, err = strconv.ParseFloat(item.GetIndex(2).MustString(), 64)
		if err != nil {
			return fmt.Errorf("invalid kline response")
		}
		// Lowest
		res[i].L, err = strconv.ParseFloat(item.GetIndex(3).MustString(), 64)
		if err != nil {
			return fmt.Errorf("invalid kline response")
		}
		// Close
		res[i].C, err = strconv.ParseFloat(item.GetIndex(4).MustString(), 64)
		if err != nil {
			return fmt.Errorf("invalid kline response")
		}
		// Volume
		res[i].V, err = strconv.ParseFloat(item.GetIndex(5).MustString(), 64)
		if err != nil {
			return fmt.Errorf("invalid kline response")
		}
	}

	p := plot.New()

	p.Title.Text = symbol
	p.X.Label.Text = "Time"
	p.Y.Label.Text = fmt.Sprintf("Price, %s", currency)
	p.X.Tick.Marker = plot.TimeTicks{Format: "2021-01-02\n15:04:05"}

	bars, err := custplotter.NewCandlesticks(res)
	if err != nil {
		return fmt.Errorf("Error while building plot")
	}

	p.Add(bars)

	// Saving graph to file, because I don't know how to work with plot bytes.
	err = p.Save(900, 400, picFileName)
	if err != nil {
		return fmt.Errorf("Error while saving plot")
	}

	photoBytes, err := ioutil.ReadFile(picFileName)
	if err != nil {
		return fmt.Errorf("Error while sending plot")
	}
	photoFileBytes := tgbotapi.FileBytes{
		Name:  "candles Graph",
		Bytes: photoBytes,
	}

	_, err = bot.Send(tgbotapi.NewPhotoUpload(chatId, photoFileBytes))

	return
}

func getBotToken() string {
	token, exists := os.LookupEnv("BOT_TOKEN")

	if !exists {
		log.Panic("Not found environment variable: BOT_TOKEN")
	}

	return token
}
