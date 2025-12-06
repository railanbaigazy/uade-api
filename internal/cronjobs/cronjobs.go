package cronjobs

import (
    "fmt"
    "log"
    

    "github.com/robfig/cron/v3"
)

// StartCronJobs запускает все cron задачи
func StartCronJobs() {
    c := cron.New()

    // --- Reminder: проверка agreements каждые 15 секунд (для примера)
    c.AddFunc("@every 15s", func() {
        log.Println("[Reminder] Проверяем agreements, которые скоро истекают...")
        // Здесь учебно: просто выводим список
        sampleAgreements := []string{"Agreement #1", "Agreement #2"}
        for _, a := range sampleAgreements {
            fmt.Println("Напоминание:", a)
        }
    })

    // --- Cleanup: удаление временных PDF каждые 30 секунд (для примера)
    c.AddFunc("@every 30s", func() {
        log.Println("[Cleanup] Удаляем временные PDF файлы...")
        // Учебно: просто выводим, без файлов
        fmt.Println("Все временные PDF удалены (симуляция)")
    })

    log.Println("Cron jobs started...")
    c.Start()
}
