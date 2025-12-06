package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"

    "github.com/railanbaigazy/uade-api/internal/app"
    "github.com/railanbaigazy/uade-api/internal/config"
    "github.com/railanbaigazy/uade-api/internal/cronjobs"
)

func main() {
    cfg := config.Load()

    // Подключение к базе (можно временно закомментировать, если нет DB)
    db, err := sqlx.Open("postgres", cfg.DBURL)
    if err != nil {
        log.Println("Не удалось подключиться к БД, продолжаем без неё:", err)
    } else {
        defer db.Close()
        if err := db.Ping(); err != nil {
            log.Println("База недоступна, продолжаем без неё:", err)
        }
    }

    // --- Запуск Cron Jobs
    cronjobs.StartCronJobs()

    // --- Настройка сервера
    a := app.New(db, cfg)
    mux := a.SetupRoutes()

    fmt.Println("Uade API running on port:", cfg.Port)
    log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}
