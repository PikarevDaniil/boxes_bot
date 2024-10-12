package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/AlexanderGrom/componenta/crypt"

	_ "github.com/go-sql-driver/mysql"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type User struct {
	id   int64
	flag int8
	pswd string
}

func main() {
	the_bot()
}

func set_tools() (*sql.DB, *tg.BotAPI, tg.UpdatesChannel) {
	// mySQL connecting
	db, err := sql.Open("mysql", "root:root@tcp(mysql:3306)/boxes")
	anti_error(err)
	// Creating tables
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS hub (id bigint, site TEXT, login TEXT, pswd TEXT)")
	anti_error(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (id bigint, flag tinyint, pswd text)")
	anti_error(err)
	// Bot Settings
	bot, err := tg.NewBotAPI("token")
	anti_error(err)
	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Update Settings
	u := tg.NewUpdate(0)
	updates := bot.GetUpdatesChan(u)

	return db, bot, updates
}

func the_bot() {
	// Variables
	db, bot, updates := set_tools()

	// Main loop
	for update := range updates {
		var err error
		current_user := open_user(db, update.FromChat().ID)
		msg := tg.NewMessage(current_user.id, "") // Create a new Message instance

		if update.Message != nil && update.Message.Text != "" {
			if !update.Message.IsCommand() {
				switch current_user.flag {
				case -2:
					current_user, _ = auth(current_user, update.Message.Text)
					msg = welcome(current_user, update.FromChat().FirstName, bot, db)
					current_user.flag = 0
				case -1:
					var ok bool
					current_user, ok = auth(current_user, update.Message.Text)
					if ok {
						msg = welcome(current_user, update.FromChat().FirstName, bot, db)
						current_user.flag = 0
					} else {
						msg.Text = "Неверно, попробуйте другой"
					}
				case 0:
					sites := check(db, update.Message.Text, current_user)
					if len(sites) == 0 {
						msg.Text = "Запись не найдена"
					} else if len(sites) == 1 {
						read(db, sites[0], bot, current_user)
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
					sites := check(db, update.Message.Text, current_user)
					if len(sites) == 0 {
						msg.Text = "Запись не найдена"

					} else if len(sites) == 1 {
						var site string
						site, err = crypt.Encrypt(sites[0], strconv.Itoa(int(current_user.id)))
						anti_error(err)
						request := "INSERT INTO hub (id, site) VALUES (?, ?)"
						_, err = db.Exec(request, current_user.id, site)
						anti_error(err)
						current_user.flag = 2
						msg.Text = "Вы точно хотите удалить запись " + sites[0] + " ?"
						msg.ReplyMarkup = tg.NewInlineKeyboardMarkup(
							tg.NewInlineKeyboardRow(
								tg.NewInlineKeyboardButtonData("Да", "Да"),
								tg.NewInlineKeyboardButtonData("Нет", "Нет"),
							),
						)
					} else {
						var kb [][]tg.InlineKeyboardButton
						msg.Text = "Найдены совпадения:"
						for _, site := range sites {
							kb = append(kb, tg.NewInlineKeyboardRow(tg.NewInlineKeyboardButtonData(site, site)))
						}
						msg.ReplyMarkup = tg.NewInlineKeyboardMarkup(kb...)
					}
				case 3, 4, 5:
					current_user, msg.Text = write(current_user, update.Message.Text, db)
				}
			} else {
				// Handle the command
				switch update.Message.Command() {
				case "start":
					log.Println("Enter")
					if current_user.flag == -2 {
						msg.Text = "Введите новый мастер-пароль"
					} else if current_user.flag == -1 {
						msg.Text = "Введите свой мастер-пароль"
					} else {
						current_user.flag = 0
						msg = welcome(current_user, update.FromChat().FirstName, bot, db)
					}
				case "add":
					switch current_user.flag {
					case -2, -1:
						msg.Text = "Откройте хранилище для начала работы"
					default:
						msg.Text = "Введите заглавие новой записи"
						current_user.flag = 3
					}
				case "del":
					switch current_user.flag {
					case -2, -1:
						msg.Text = "Откройте хранилище для начала работы"
					default:
						current_user.flag = 1
						if check_keyboard(db, current_user) {
							msg.Text = "Выберите запись"
							msg.ReplyMarkup = build(db, current_user)
						} else {
							msg.Text = "У вас нет ни одной записи"
						}
					}
				case "find":
					switch current_user.flag {
					case -2, -1:
						msg.Text = "Откройте хранилище для начала работы"
					default:
						current_user.flag = 0
						if check_keyboard(db, current_user) {
							msg.Text = "Выберите запись..."
							msg.ReplyMarkup = build(db, current_user)
						} else {
							msg.Text = "У вас нет ни одной записи"
						}
					}
				case "help":
					data, err := os.ReadFile("help.txt")
					anti_error(err)
					msg.Text = string(data)
				case "exit":
					hash, err := hash_pswd(current_user.pswd)
					anti_error(err)
					log.Println("Exit")
					log.Println("hash to write: ", hash)
					current_user.pswd = hash
					current_user.flag = -1
					msg.Text = "Хранилище закрыто"
				default:
					msg.Text = "Я не знаю такой команды"
				}
			}
			// Send the Message
			_, err := bot.Send(msg)
			anti_error(err)

		} else if update.CallbackQuery != nil {

			callback := tg.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
			_, err := bot.Request(callback)
			anti_error(err)

			if current_user.flag == 0 {
				sites := check(db, update.CallbackQuery.Data, current_user)
				if len(sites) == 0 {
					msg.Text = "Запись не найдена"
					_, err = bot.Send(msg)
					anti_error(err)
				} else {
					read(db, sites[0], bot, current_user)
				}
			} else if current_user.flag == 1 {
				sites := check(db, update.CallbackQuery.Data, current_user)
				if len(sites) == 0 {
					msg.Text = "Запись не найдена"
				} else {
					var site string
					site, err = crypt.Encrypt(sites[0], strconv.Itoa(int(current_user.id)))
					anti_error(err)
					_, err = db.Exec("INSERT INTO hub (id, site) VALUES (?, ?)", current_user.id, site)
					anti_error(err)
					current_user.flag = 2
					msg.Text = "Вы точно хотите удалить запись " + sites[0] + " ?"
					msg.ReplyMarkup = tg.NewInlineKeyboardMarkup(
						tg.NewInlineKeyboardRow(
							tg.NewInlineKeyboardButtonData("Да", "Да"),
							tg.NewInlineKeyboardButtonData("Нет", "Нет"),
						),
					)
				}
				_, err = bot.Send(msg)
				anti_error(err)
			} else if current_user.flag == 2 && update.CallbackQuery.Data == "Да" {
				current_user.flag = 1
				site, err := crypt.Decrypt(read_site(db, current_user), strconv.Itoa(int(current_user.id)))
				anti_error(err)
				delete(db, site, current_user)
				msg.Text = "Запись Удалена"
				_, err = bot.Send(msg)
				anti_error(err)
			} else if current_user.flag == 2 && update.CallbackQuery.Data == "Нет" {
				current_user.flag = 1
				msg.Text = "Удалние Отменено"
				_, err = bot.Send(msg)
				anti_error(err)
			}
		} else {
			sticker := tg.NewSticker(current_user.id, tg.FileID("CAACAgIAAxkBAAEMk7VmqmwGNzv67g2mcRrawivei0Q16wACN0IAAiJIEEtr9OEdF85NeDUE"))
			_, err := bot.Send(sticker)
			anti_error(err)
		}
		// Update the user data with new user's flag
		close_user(db, current_user)
	}
}

func write(user User, text string, db *sql.DB) (User, string) {
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

func read(db *sql.DB, site string, bot *tg.BotAPI, user User) {
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

func delete(db *sql.DB, site string, user User) {
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
		log.Println(site, outsite)
		if strings.EqualFold(site, outsite) {
			_, err = db.Exec("DELETE FROM hub WHERE id =? AND site =?", user.id, crypt_out)
			anti_error(err)
		}
	}
}

func build(db *sql.DB, user User) tg.InlineKeyboardMarkup {
	log.Println("Building Keyboard...")
	rows, err := db.Query("SELECT site FROM hub WHERE id =?", user.id)
	anti_error(err)

	defer rows.Close()
	var sites []string
	var keyboard [][]tg.InlineKeyboardButton
	for rows.Next() {
		var site string
		err := rows.Scan(&site)
		anti_error(err)
		site, err = crypt.Decrypt(site, user.pswd)
		anti_error(err)
		if !in_list(sites, site) {
			sites = append(sites, site)
			keyboard = append(keyboard, tg.NewInlineKeyboardRow(tg.NewInlineKeyboardButtonData(site, site)))
		}
	}
	return tg.NewInlineKeyboardMarkup(keyboard...)
}

func check_keyboard(db *sql.DB, user User) bool {
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

// TO RENAME
func check(db *sql.DB, site string, user User) []string {
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
		matched := strings.Contains(strings.ToLower(outsite), strings.ToLower(site))
		if matched && !in_list(out, outsite) {
			out = append(out, outsite)
		}
	}
	return out
}

func in_list(list []string, e string) bool {
	for _, a := range list {
		if a == e {
			return true
		}
	}
	return false
}

func anti_error(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func open_user(db *sql.DB, id int64) User {
	var user User
	row := db.QueryRow("SELECT flag, pswd FROM users WHERE id =?", id)
	user.id = id
	err := row.Scan(&user.flag, &user.pswd)
	if err == sql.ErrNoRows {
		_, err = db.Exec("INSERT INTO users (id, flag) VALUES (?, ?)", id, -2)
		anti_error(err)
		return User{id, -2, ""}
	}
	anti_error(err)

	return user
}

func close_user(db *sql.DB, user User) {
	log.Println("pswd to write: ", user.pswd)
	_, err := db.Exec("UPDATE users SET flag =?, pswd =? WHERE id =?", user.flag, user.pswd, user.id)
	anti_error(err)
}

func read_site(db *sql.DB, user User) string {
	var out string
	row, err := db.Query("SELECT site FROM hub WHERE id =? AND login IS NULL", user.id)
	anti_error(err)
	for row.Next() {
		err := row.Scan(&out)
		anti_error(err)
	}
	return out
}

func auth(user User, pswd string) (User, bool) {
	var is_correct bool
	if user.flag == -2 {
		user.pswd = pswd
		is_correct = true
	} else if user.flag == -1 {
		if check_pswd(pswd, user) {
			log.Println("Correct pswd")
			user.pswd = pswd
			is_correct = true
		} else {
			log.Println("Incorrent pswd")
			is_correct = false
		}
	}
	return user, is_correct
}

func welcome(user User, name string, bot *tg.BotAPI, db *sql.DB) tg.MessageConfig {
	msg := tg.NewMessage(user.id, "")
	msg.Text = "Добро пожаловать, " + name + "!"
	bot.Send(msg)
	if check_keyboard(db, user) {
		msg.Text = "Выберите запись..."
		msg.ReplyMarkup = build(db, user)
	} else {
		msg.Text = "У вас нет ни одной записи"
	}
	return msg
}

func check_pswd(pswd string, user User) bool {
	log.Println("hash to check: ", user.pswd)
	log.Println("pswd to check: ", pswd)
	return bcrypt.CompareHashAndPassword([]byte(user.pswd), []byte(pswd)) == nil
}

func hash_pswd(pswd string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pswd), bcrypt.DefaultCost)
	return string(bytes), err
}
