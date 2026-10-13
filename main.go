package main

import (
	"fmt"
	"log"
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
	Databases []struct {
		Name         string `yaml:"name"`
		DBhost       string `yaml:"host"`
		Port         int    `yaml:"port"`
		User         string `yaml:"user"`
		Password     string `yaml:"password"`
		DatabaseName string `yaml:"database"`
		KeepForDays  int    `yaml:"days"`
	} `yaml:"databases"`
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

	//Вызов pg_dump
	for _, db := range cfg.Databases {
		if err := os.MkdirAll("./backups/"+db.DatabaseName, 0750); err != nil {
			log.Fatalf("Error creating a directory: %v", err)
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
			log.Printf("Error creating a backup: %v", err)
		} else {
			log.Printf("[%s] SUCCESS: сохранён в %s", db.Name, backupPath)
		}
	}
}
