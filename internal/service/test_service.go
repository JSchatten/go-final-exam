package service

import "gopkg.in/telebot.v3"

var (
// MenuTest = &telebot.ReplyMarkup{ResizeKeyboard: true}
// // Paginator = &telebot.ReplyMarkup{}
// SelectorTest = &telebot.ReplyMarkup{}

// BtnStartTest = MenuInBot.Text("start")
// btnHelpTest  = MenuInBot.Text("help")
// btnFindTest  = MenuInBot.Text("Find")
)

func (b *BotService) HandleTest(c telebot.Context) error {
	return c.Reply("Test message!")
}
