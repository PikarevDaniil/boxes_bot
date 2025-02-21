package main

// imports
import (
	"database/sql"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/AlexanderGrom/componenta/crypt"

	_ "github.com/go-sql-driver/mysql"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// User's structure
type User struct {
	id   int64
	flag int8
	pswd string
}

var db, bot, updates = set_tools()
var users = make(map[int64]string)

// settings
func set_tools() (*sql.DB, *tg.BotAPI, tg.UpdatesChannel) {
	// mySQL connecting
	db, err := sql.Open("mysql", "root:"+os.Getenv("MYSQL_PASSWORD")+"@tcp(mysql:3306)/boxes")
	anti_error(err)
	// Creating tables
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS hub (n int auto_increment primary key, id bigint, site TEXT, login TEXT, pswd TEXT)")
	anti_error(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (id bigint, flag tinyint, pswd text)")
	anti_error(err)
	// Bot Settings
	bot, err := tg.NewBotAPI(os.Getenv("TELEGRAM_APITOKEN"))
	anti_error(err)
	bot.Debug = false

	// Update Settings
	u := tg.NewUpdate(0)
	updates := bot.GetUpdatesChan(u)

	return db, bot, updates
}

// check if element in list
func in_list(list []string, e string) bool {
	for _, a := range list {
		if a == e {
			return true
		}
	}
	return false
}

// errors
func anti_error(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// interence point
func main() {
	for update := range updates {
		if update.FromChat() == nil {
			log.Println("Ошибка: update.FromChat() вернул nil. Пропускаем обновление.")
			continue
		}
		current_user := open_user(update.FromChat().ID)
		msg := tg.NewMessage(current_user.id, "") // Create a new Message instance

		if update.Message != nil && update.Message.Text != "" {
			if !update.Message.IsCommand() {
				msg, current_user = handle_text(current_user, msg, update)
			} else {
				msg, current_user = handle_command(current_user, msg, update)
			}
			_, err := bot.Send(msg)
			anti_error(err)

		} else if update.CallbackQuery != nil {
			msg, current_user = handle_kd(current_user, msg, update)
			bot.Send(msg)
		} else {
			sticker := tg.NewSticker(current_user.id, tg.FileID("CAACAgIAAxkBAAENep1ngqeRXehcx8gz_8Ma2tPoKcy9uAACjicAAnJ6-UgOlPQwPgxYlzYE"))
			_, err := bot.Send(sticker)
			anti_error(err)
		}
		// Update the user data with new user's flag
		close_user(current_user)
	}
}

// handle text msg
func handle_text(user User, msg tg.MessageConfig, update tg.Update) (tg.MessageConfig, User) {
	switch user.flag {
	case -2:
		user, _ = auth(user, update.Message.Text)
		msg = welcome(user, update.FromChat().FirstName)
		user.flag = 0
	case -1:
		var ok bool
		user, ok = auth(user, update.Message.Text)
		if ok {
			msg = welcome(user, update.FromChat().FirstName)
			user.flag = 0
		} else {
			msg.Text = "Неверно, попробуйте другой"
		}
	case 0:
		sites := site_list(user, update.Message.Text, 1)
		if len(sites) == 0 {
			msg.Text = "Запись не найдена"
		} else if len(sites) == 1 {
			read(sites[0], user)
			msg.Text = "..."
		} else {
			var kb [][]tg.InlineKeyboardButton
			msg.Text = "Найдены совпадения:"
			for _, site := range sites {
				kb = append(kb, tg.NewInlineKeyboardRow(tg.NewInlineKeyboardButtonData(site, site)))
			}
			msg.ReplyMarkup = tg.NewInlineKeyboardMarkup(kb...)
		}
	case 1:
		msg, user.flag = to_delete(msg, user, update.Message.Text, 1)
	case 2:
		if update.Message.Text == "Да" {
			site, err := crypt.Decrypt(read_site(user), user.pswd)
			anti_error(err)
			delete_data(site, user)
			msg.Text = "Запись Удалена"
		} else {
			msg.Text = "Удалние Отменено"
		}
		user.flag = 1
	case 3, 4, 5:
		user, msg.Text = write(user, update.Message.Text)
	case 6:
		change_pswd(update.Message.Text, user.id)
		msg.Text = "Вы успешно сменили мастер-пароль!"
		user.flag = 0
	}
	return msg, user
}

// handle commands
func handle_command(user User, msg tg.MessageConfig, update tg.Update) (tg.MessageConfig, User) {
	switch update.Message.Command() {
	case "start":
		if user.flag == -2 {
			msg.Text = "Введите мастер-пароль, которым будут шифроваться ваши данные"
		} else if user.flag == -1 {
			msg.Text = "Введите свой мастер-пароль"
		} else {
			user.flag = 0
			msg = welcome(user, update.FromChat().FirstName)
		}
	case "add":
		switch user.flag {
		case -2:
			msg.Text = "Сначала создайте сейф"
		case -1:
			msg.Text = "Сначала откройте сейф"
		default:
			msg.Text = "Введите заглавие новой записи"
			user.flag = 3
		}
	case "del":
		switch user.flag {
		case -2:
			msg.Text = "Сначала создайте сейф"
		case -1:
			msg.Text = "Сначала откройте сейф"
		default:
			user.flag = 1
			if check_keyboard(user) {
				msg.Text = "Выберите запись"
				msg.ReplyMarkup = build(user)
			} else {
				msg.Text = "Сейф пуст..."
			}
		}
	case "find":
		switch user.flag {
		case -2:
			msg.Text = "Сначала создайте сейф"
		case -1:
			msg.Text = "Сначала откройте сейф"
		default:
			user.flag = 0
			if check_keyboard(user) {
				msg.Text = "Выберите запись..."
				msg.ReplyMarkup = build(user)
			} else {
				msg.Text = "Ваш сейф пуст..."
			}
		}
	case "help":
		data, err := os.ReadFile("help.txt")
		anti_error(err)
		msg.Text = string(data)
	case "exit":
		switch user.flag {
		case -2:
			msg.Text = "Сначала создайте сейф"
		case -1:
			msg.Text = "Сейф уже закрыт"
		default:
			delete(users, user.id)
			user.flag = -1
			msg.Text = "Сейф успешно закрыт"
		}
	case "change":
		if user.flag == -2 {
			msg.Text = "Сначала создайте сейф"
		} else if user.flag == -1 {
			msg.Text = "Сначала откройте сейф"
		} else {
			msg.Text = "Введите новый мастер-пароль"
			user.flag = 6
		}

	default:
		msg.Text = "Я не знаю такой команды"
	}
	return msg, user
}

// hanle inline keyboards upates
func handle_kd(user User, msg tg.MessageConfig, update tg.Update) (tg.MessageConfig, User) {
	callback := tg.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
	_, err := bot.Request(callback)
	anti_error(err)

	if user.flag == 0 {
		sites := site_list(user, update.CallbackQuery.Data, 2)
		if len(sites) == 0 {
			msg.Text = "Запись не найдена"
		} else {
			read(sites[0], user)
		}
	} else if user.flag == 1 {
		msg, user.flag = to_delete(msg, user, update.CallbackQuery.Data, 2)
	} else if user.flag == 2 && update.CallbackQuery.Data == "Да" {
		user.flag = 1
		site, err := crypt.Decrypt(read_site(user), user.pswd)
		anti_error(err)
		delete_data(site, user)
		msg.Text = "Запись Удалена"
	} else if user.flag == 2 && update.CallbackQuery.Data == "Нет" {
		user.flag = 1
		msg.Text = "Удалние Отменено"
	}
	return msg, user
}

// write data
func write(user User, text string) (User, string) {
	out := "..."
	var err error
	switch user.flag {
	case 3:
		_, err = db.Exec("DELETE FROM hub WHERE id =? AND pswd IS NULL", user.id)
		anti_error(err)
		site, err := crypt.Encrypt(text, user.pswd)
		anti_error(err)
		_, err = db.Exec("INSERT INTO hub (id, site) VALUES (?, ?)", user.id, site)
		anti_error(err)
		out = "Введите логин..."
		user.flag = 4
	case 4:
		login, err := crypt.Encrypt(text, user.pswd)
		anti_error(err)
		_, err = db.Exec("UPDATE hub SET login =? WHERE id =? AND login IS NULL", login, user.id)
		anti_error(err)
		out = "Введите пароль..."
		user.flag = 5
	case 5:
		pswd, err := crypt.Encrypt(text, user.pswd)
		anti_error(err)
		_, err = db.Exec("UPDATE hub SET pswd =? WHERE id =? AND pswd IS NULL", pswd, user.id)
		anti_error(err)
		out = "Записано"
		user.flag = 0
	}
	return user, out
}

// read data
func read(site string, user User) {
	rows, err := db.Query("SELECT site, login, pswd FROM hub WHERE id =?", user.id)
	anti_error(err)

	defer rows.Close()
	var out string
	var outsite string
	var login string
	var pswd string

	for rows.Next() {
		err := rows.Scan(&outsite, &login, &pswd)
		anti_error(err)
		outsite, err = crypt.Decrypt(outsite, user.pswd)
		anti_error(err)
		if strings.EqualFold(site, outsite) {
			login, err = crypt.Decrypt(login, user.pswd)
			anti_error(err)
			pswd, err = crypt.Decrypt(pswd, user.pswd)
			anti_error(err)
			out = outsite + "\n" + "Логин: " + login + "\n" + "Пароль: " + pswd
			msg := tg.NewMessage(user.id, out)
			bot.Send(msg)
		}
	}
}

// ask user if he really wants to remove data
func to_delete(msg tg.MessageConfig, user User, site string, f uint8) (tg.MessageConfig, int8) {
	var err error
	flag := user.flag
	sites := site_list(user, site, f)
	if len(sites) == 0 {
		msg.Text = "Запись не найдена"

	} else if len(sites) == 1 {
		var site string
		site, err = crypt.Encrypt(sites[0], user.pswd)
		anti_error(err)
		request := "INSERT INTO hub (id, site) VALUES (?, ?)"
		_, err = db.Exec(request, user.id, site)
		anti_error(err)
		msg.Text = "Вы точно хотите удалить запись " + sites[0] + " ?"
		msg.ReplyMarkup = tg.NewInlineKeyboardMarkup(
			tg.NewInlineKeyboardRow(
				tg.NewInlineKeyboardButtonData("Да", "Да"),
				tg.NewInlineKeyboardButtonData("Нет", "Нет"),
			),
		)
		flag = 2
	} else {
		var kb [][]tg.InlineKeyboardButton
		msg.Text = "Найдены совпадения:"
		for _, site := range sites {
			kb = append(kb, tg.NewInlineKeyboardRow(tg.NewInlineKeyboardButtonData(site, site)))
		}
		msg.ReplyMarkup = tg.NewInlineKeyboardMarkup(kb...)
	}
	return msg, flag
}

// remove data
func delete_data(site string, user User) {
	rows, err := db.Query("SELECT site FROM hub WHERE id =?", user.id)
	anti_error(err)
	defer rows.Close()
	for rows.Next() {
		var outsite string
		var crypt_out string
		err := rows.Scan(&crypt_out)
		anti_error(err)
		outsite, err = crypt.Decrypt(crypt_out, user.pswd)
		anti_error(err)
		if strings.EqualFold(site, outsite) {
			_, err = db.Exec("DELETE FROM hub WHERE id =? AND site =?", user.id, crypt_out)
			anti_error(err)
		}
	}
}

// read name of data to remove
func read_site(user User) string {
	var out string
	row, err := db.Query("SELECT site FROM hub WHERE id =? AND login IS NULL", user.id)
	anti_error(err)
	for row.Next() {
		err := row.Scan(&out)
		anti_error(err)
	}
	return out
}

// build keyboard with user's data
func build(user User) tg.InlineKeyboardMarkup {
	var kb [][]tg.InlineKeyboardButton
	for _, site := range site_list(user, "", 1) {
		kb = append(kb, tg.NewInlineKeyboardRow(tg.NewInlineKeyboardButtonData(site, site)))
	}
	return tg.NewInlineKeyboardMarkup(kb...)
}

// site list for finding
func site_list(user User, site string, f uint8) []string {
	var matched bool
	_, err := db.Exec("DELETE FROM hub WHERE id =? AND pswd IS NULL", user.id)
	anti_error(err)
	rows, err := db.Query("SELECT site FROM hub WHERE id =?", user.id)
	var outsite string
	var out []string
	anti_error(err)

	defer rows.Close()
	for rows.Next() {
		rows.Scan(&outsite)
		outsite, err = crypt.Decrypt(outsite, user.pswd)
		anti_error(err)
		if f == 1 {
			matched = strings.Contains(strings.ToLower(outsite), strings.ToLower(site))
		} else {
			matched = outsite == site
		}

		if matched && !in_list(out, outsite) {
			out = append(out, outsite)
		}
	}
	return out
}

// check if the user has any data
func check_keyboard(user User) bool {
	_, err := db.Exec("DELETE FROM hub WHERE id =? AND pswd IS NULL", user.id)
	anti_error(err)
	is_rows, err := db.Query("SELECT COUNT(site) FROM hub WHERE id =?", user.id)
	anti_error(err)

	defer is_rows.Close()
	var out int
	for is_rows.Next() {
		is_rows.Scan(&out)
	}
	return out > 0
}

// get user's paramethers
func open_user(id int64) User {
	var user User
	row := db.QueryRow("SELECT flag, pswd FROM users WHERE id =?", id)
	user.id = id
	err := row.Scan(&user.flag, &user.pswd)
	if err == sql.ErrNoRows {
		_, err = db.Exec("INSERT INTO users (id, flag, pswd) VALUES (?, ?, ?)", id, -2, "")
		anti_error(err)
		return User{id, -2, ""}
	}
	if _, ok := users[user.id]; !ok && user.flag != -2 {
		user.flag = -1
	} else {
		user.pswd = users[user.id]
	}
	anti_error(err)

	return user
}

// set user's paramethers
func close_user(user User) {
	_, err := db.Exec("UPDATE users SET flag =? WHERE id =?", user.flag, user.id)
	anti_error(err)
}

// welcome, user!
func welcome(user User, name string) tg.MessageConfig {
	msg := tg.NewMessage(user.id, "")
	msg.Text = "Добро пожаловать, " + name + "!"
	bot.Send(msg)
	if check_keyboard(user) {
		msg.Text = "Выберите запись..."
		msg.ReplyMarkup = build(user)
	} else {
		msg.Text = "У вас нет ни одной записи"
	}
	return msg
}

// authorisation
func auth(user User, pswd string) (User, bool) {
	var is_correct bool
	if user.flag == -2 {
		users[user.id] = pswd
		hash, err := hash_pswd(pswd)
		anti_error(err)
		_, err = db.Exec("UPDATE users SET pswd =? WHERE id =?", hash, user.id)
		anti_error(err)
		is_correct = true
	} else if user.flag == -1 {
		if check_pswd(pswd, user) {
			users[user.id] = pswd
			user.pswd = pswd
			is_correct = true
		} else {
			is_correct = false
		}
	}
	return user, is_correct
}

// check if user entered right master-pswd
func check_pswd(pswd string, user User) bool {
	return bcrypt.CompareHashAndPassword([]byte(user.pswd), []byte(pswd)) == nil
}

// hash user's master-pswd
func hash_pswd(pswd string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pswd), bcrypt.DefaultCost)
	return string(bytes), err
}

func change_pswd(new_master_pswd string, id int64) {
	var n int
	var site string
	var login string
	var pswd string
	query := "UPDATE hub SET site =?, login =?, pswd =? WHERE n =?"
	to_write, err := hash_pswd(new_master_pswd)
	rows, err := db.Query("SELECT n, site, login, pswd FROM hub WHERE id =?", id)
	anti_error(err)
	defer rows.Close()
	for rows.Next() {

		rows.Scan(&n, &site, &login, &pswd)
		new_site, err := crypt.Decrypt(site, users[id])
		anti_error(err)
		new_login, err := crypt.Decrypt(login, users[id])
		anti_error(err)
		new_pswd, err := crypt.Decrypt(pswd, users[id])
		anti_error(err)

		new_site, err = crypt.Encrypt(new_site, new_master_pswd)
		anti_error(err)
		new_login, err = crypt.Encrypt(new_login, new_master_pswd)
		anti_error(err)
		new_pswd, err = crypt.Encrypt(new_pswd, new_master_pswd)
		anti_error(err)

		db.Exec(query, new_site, new_login, new_pswd, n)
	}
	db.Exec("UPDATE users SET pswd =? WHERE id =?", to_write, id)
	users[id] = new_master_pswd
}
