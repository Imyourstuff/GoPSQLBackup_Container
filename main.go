package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Defaults struct {
		CronTab     string `yaml:"cron"`
		KeepForDays int    `yaml:"days"`
	} `yaml:"defaults"`
	Databases []Database `yaml:"databases"`
}

type Database struct {
	Name         string `yaml:"name"`
	DBhost       string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	DatabaseName string `yaml:"database"`
	KeepForDays  int    `yaml:"days"`
}

func run(db Database) error {
	if err := os.MkdirAll("./backups/"+db.DatabaseName, 0750); err != nil {
		return fmt.Errorf("Error creating a directory: %v", err)
	}
	backupPath := "./backups/" + db.DatabaseName + "/" + db.DatabaseName + time.Now().Format("2006-01-02_15-04-05") + ".dump"

	log.Printf("DEBUG: Host:'%s', Port=%d, DB='%s', User='%s'",
		db.DBhost, db.Port, db.DatabaseName, db.User)

	//Проверка на значения
	if db.Port == 0 {
		db.Port = 5432
	}
	if db.User == "" {
		db.User = "postgres"
	}

	//Готовим команду
	cmd := exec.Command("pg_dump",
		"-h", db.DBhost,
		"-p", strconv.Itoa(db.Port),
		"-U", db.User,
		"-d", db.DatabaseName,
		"-F", "c",
		"-f", backupPath)
	cmd.Stderr = os.Stderr //Для отлова ошибок pg_backup
	//Тут лоханулся что приращиваем в арги самой cmd и он трёт из OS ENV
	cmd.Env = append(os.Environ(), "PGPASSWORD="+db.Password)
	if err := cmd.Run(); err != nil {
		os.Remove(backupPath)
		return fmt.Errorf("Error creating a backup: %v", err)
	} else {
		log.Printf("[%s] SUCCESS: сохранён в %s", db.Name, backupPath)
	}
	return nil
}

func main() {
	//Импорт конфига и дереализация
	file, err := os.Open("backup-conf.yaml")
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}
	defer file.Close()

	dec := yaml.NewDecoder(file)
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		log.Fatalf("Error decoding YAML: %v", err)
	}
	fmt.Printf("Config imported from backup-conf.yaml, crontab set to: %s\n", cfg.Defaults.CronTab)

	http.HandleFunc("/backup", func(w http.ResponseWriter, r *http.Request) {
		var found bool = false
		name := r.URL.Query().Get("db")
		for _, db := range cfg.Databases {
			if name == "" || db.DatabaseName == name || db.Name == name {
				found = true
				go func(database Database) {
					if err := run(database); err != nil {
						log.Printf("[%s] Oshibka: %v", database.Name, err)
					}
				}(db)
			}
		}

		if !found && name != "" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "База '%s' не найдена в backup-conf.yaml!!!!!!!!\n", name)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		if name == "" {
			fmt.Fprintln(w, "ВСЁ БЕКАПИМ")
		} else {
			fmt.Fprintf(w, "ОДНУ БАЗУ СБЕКАПИЛИ: %s\n", name)
		}
	})

	//Вызов pg_dump теперь в run, при первом запуске бекапим всё?
	//Так надо или не надо...
	for _, db := range cfg.Databases {
		go func(database Database) {
			if err := run(database); err != nil {
				log.Printf("[%s] Первый бекап не сделан ((: %v", database.Name, err)
			}
		}(db)
	}

	port := ":8080"
	log.Printf("СЛУШАЕМ ПОРТ %s (/backup?db=имя_базы)...", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Ошибка HTTP сервера: %v", err)
	}
}
