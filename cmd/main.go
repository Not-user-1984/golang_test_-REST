package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/utils"
	"gopkg.in/reform.v1"
	_ "github.com/go-sql-driver/mysql"
)

// News структура для хранения данных о новостях
type News struct {
	ID      int64  `reform:"id,pk"`
	Title   string `reform:"title"`
	Content string `reform:"content"`
}

// NewsCategories структура для хранения связей новостей и категорий
type NewsCategories struct {
	NewsID     int64 `reform:"news_id,pk"`
	CategoryID int64 `reform:"category_id,pk"`
}

var (
	db  *reform.DB
	err error
)

func initDB() {
	// Получаем параметры подключения из переменных окружения
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")

	// Формируем строку подключения
	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)

	// Инициализируем базу данных с использованием connection pool
	db, err = reform.NewDB("mysql", connectionString, reform.NewPrintfLogger(log.Printf))
	if err != nil {
		log.Fatal(err)
	}

	// Создаем таблицы, если они не существуют
	if err := db.CreateTableIfNotExists(new(News), new(NewsCategories)); err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()

	app := fiber.New()

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// Ручка для изменения новости по Id
	app.Post("/edit/:Id", func(c *fiber.Ctx) error {
		id := utils.ToInt64(c.Params("Id"))
		news := new(News)
		if err := c.BodyParser(news); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		// Найдем новость по Id
		existingNews, err := db.FindByPrimaryKeyFrom(new(News), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal Server Error"})
		}

		if existingNews == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "News not found"})
		}

		// Обновим данные новости, если они заданы
		if news.Title != "" {
			existingNews.(*News).Title = news.Title
		}
		if news.Content != "" {
			existingNews.(*News).Content = news.Content
		}

		// Сохраняем изменения
		if err := db.Save(existingNews); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal Server Error"})
		}

		return c.Status(fiber.StatusOK).JSON(existingNews)
	})

	// Ручка для получения списка новостей
	app.Get("/list", func(c *fiber.Ctx) error {
		newsList, err := db.SelectAllFrom(new(News))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal Server Error"})
		}

		// Преобразуем список новостей в нужный формат
		var formattedNews []fiber.Map
		for _, news := range newsList {
			newsMap := fiber.Map{
				"Id":      news.(*News).ID,
				"Title":   news.(*News).Title,
				"Content": news.(*News).Content,
			}
			formattedNews = append(formattedNews, newsMap)
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"Success": true, "News": formattedNews})
	})

	// Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(app.Listen(":" + port))
}
