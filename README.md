# Простейший телеграм бот на Golang

### Важно для запуска

1. Создать бота через BotFather, получить токен
2. Перед запуском создать в директории проекта файл .env в котором прописать:

```
BOT_TOKEN=<Your bot token>
```

### Команды

`ADD <SYMBOL> <AMOUNT>` - добавить криптовалюту в кошелек, SYMBOL - валюта, например BTC, AMOUNT - сумма в формате X.XXX

`SUB <SYMBOL> <AMOUNT>` - уменьшить баланс кошелька по криптовалюте, SYMBOL - валюта, например BTC, AMOUNT - сумма в формате X.XXX, если баланс становится равным 0, валюта удаляется

`DEL <SYMBOL>` - удалить валюту из кошелька, SYMBOL - валюта, например BTC

`SHOW` - отобразить список валют с балансами и итоговый баланс

### Дополнительно

Попробовал выводить график

`GRAPH <SYMBOL>` - вывести график свечей выбранной криптовалюты, SYMBOL за последние сутки с интервалом 30 мин