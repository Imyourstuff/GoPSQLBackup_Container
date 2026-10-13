#!/bin/bash
set -eo pipefail

NOW=$(date +"%Y-%m-%d_%H-%M-%S")
CONFIG="/etc/backup/backup-conf.yaml"
TARGET_NAME="$1" #первый аргумент для работы только с одной бд

if [ ! -f "$CONFIG" ]; then
    echo "КОНФИГ Г Д Е?"
    exit 1
fi

# чтобы FOR работал в DBS должны быть через ПРОБЕЛЫ
# парсер впадлу пока
# upd пришлось делать парсер но для yaml я не ебу как им пользоваться
TOTAL_DBS=$(yq '.databases | length' "$CONFIG")
DAYS_LEFT=$(yq '.defaults.retention_days // 7' "$CONFIG")

for ((i=0; i<TOTAL_DBS; i++)) do
    NAME=$(yq ".databases[$i].name" "$CONFIG")
    HOST=$(yq ".databases[$i].host" "$CONFIG")
    PORT=$(yq ".databases[$i].port // 5432" "$CONFIG")
    USER=$(yq ".databases[$i].user // \"postgres\"" "$CONFIG")
    PASS=$(yq ".databases[$i].password" "$CONFIG")
    DB=$(yq ".databases[$i].database" "$CONFIG")
    RETENTION=$(yq ".databases[$i].retention_days // $DAYS_LEFT" "$CONFIG")
    # какой ужас надо было писать на go

    if [ -n "$TARGET_NAME" ] && [ "$TARGET_NAME" != "$NAME" ]; then
        continue
    fi

    DEST_DIR="/backups/${NAME}"
    mkdir -p "$DEST_DIR"
    FILE="${DEST_DIR}/${NAME}_${NOW}.dump"

    set +e
    PGPASSWORD="$PASS" pg_dump \
        -h "$HOST" \
        -p "$PORT" \
        -U "$USER" \
        -d "$DB" \
        -F c \
        -f "$FILE"
    STATUS=$?
    set -e

    if [ $STATUS -ne 0 ] || [ ! -s "$FILE" ] || ! pg_restore -l "$FILE" > /dev/null 2>&1; then
        echo "FATAL: Сбой при создании бэкапа для $NAME!"
        [ -f "$FILE" ] && rm -f "$FILE"
        continue
    fi

    SIZE=$(du -h "$FILE" | cut -f1)
    echo "OK: $FILE сохранён ($SIZE)"

    # Ротация файлов конкретного хоста
    if [ -n "$RETENTION" ] && [ "$RETENTION" -gt 0 ]; then
        find "$DEST_DIR" -maxdepth 1 -name "${NAME}_*.dump" -mtime +"$RETENTION" -delete
    fi

done
