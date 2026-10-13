#!/usr/bin/env bash

#* * * * * *
#| | | | | | 
#| | | | | +-- Year              (range: 1900-3000)
#| | | | +---- Day of the Week   (range: 1-7, 1 standing for Monday)
#| | | +------ Month of the Year (range: 1-12)
#| | +-------- Day of the Month  (range: 1-31)
#| +---------- Hour              (range: 0-23)
#+------------ Minute            (range: 0-59)

# :- если нет из env то ставим дефолт, чичас 5:30 ???
CONFIG="/etc/backup/backup-conf.yaml"

if [ ! -f "$CONFIG" ]; then
    echo "FATAL: Файл $CONFIG не прочитан"
    exit 1
fi

CRON_SCHEDULE=$(yq '.defaults.cron // "0 3 * * *"' "$CONFIG")
echo "Расписание для бекапов по cron: $CRON_SCHEDULE"

# Регистрируем задачу в crontab
echo "${CRON_SCHEDULE} /usr/local/bin/backup >> /var/log/backup.log 2>&1" > /etc/crontabs/root

exec crond -f -l 8