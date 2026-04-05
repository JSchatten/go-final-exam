package service

import (
	"fmt"
	"log"

	"github.com/JSchatten/go-final-exam/internal/models"
	"gopkg.in/telebot.v3"
)

// HandleStart обрабатывает команду /start.
func (b *Bot) HandleStart(c telebot.Context) error {
	user := c.Sender()

	// Проверяем, существует ли пользователь
	existingUser, err := b.UserRepo.FindByTelegramID(user.ID)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		// Продолжаем, даже если ошибка - возможно, БД временно недоступна
	}

	// Создаём модель для сохранения/обновления
	userDB := &models.User{
		TelegramID: user.ID,
		Username:   &user.Username,
		FirstName:  &user.FirstName,
		LastName:   &user.LastName,
		// CreatedAt:  time.Now().UTC(),
		// UpdatedAt: time.Now().UTC(),
	}

	var message string

	if existingUser == nil {
		err = b.UserRepo.CreateIfNotExists(userDB) // Пользователя нет - регистрируем нового
		if err != nil {
			log.Printf("Failed to save new user: %v", err) // всё равно отправим приветствие
		}
		message = fmt.Sprintf("Добро пожаловать, %s!\nТы успешно зарегистрирован.", user.FirstName)
	} else {
		// Обновляем данные и время последнего визита
		userDB.ID = existingUser.ID // нужно для обновления
		err = b.UserRepo.Update(userDB)
		if err != nil {
			log.Printf("Failed to update user: %v", err)
		}
		message = fmt.Sprintf("С возвращением, %s!\nРад снова тебя видеть.", user.FirstName)
	}

	return c.Send(message)
}
