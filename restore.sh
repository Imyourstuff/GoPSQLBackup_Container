#!/bin/bash
set -eo pipefail

CONFIG="/etc/backup/backup-conf.yaml"
NAME="$1"
DUMP_FILE="$2"

if [ -z "$NAME" ]; then
    echo "Usecase: restore <name_из_yaml> [dump_filename]"
    echo "Доступные базы в конфиге yaml:"
    yq '.databases[].name' "$CONFIG"
    exit 1
fi

ENTRY=$(yq ".databases[] | select(.name == \"$NAME\")" "$CONFIG")

if [ -z "$ENTRY" ]; then
    echo "ERROR: База '$NAME' не найдена в $CONFIG"
    exit 1
fi

HOST=$(echo "$ENTRY" | yq '.host')
PORT=$(echo "$ENTRY" | yq '.port // 5432')
USER=$(echo "$ENTRY" | yq '.user // "postgres"')
PASS=$(echo "$ENTRY" | yq '.password')
DB=$(echo "$ENTRY" | yq '.database')

DEST_DIR="/backups/${NAME}"

if [ -z "$DUMP_FILE" ]; then
    DUMP_FILE=$(find "$DEST_DIR" -maxdepth 1 -name "${NAME}_*.dump" -type f | sort -r | head -n 1)
    if [ -z "$DUMP_FILE" ]; then
        echo "ERROR: Бэкапов для '$NAME' в $DEST_DIR нет!"
        exit 1
    fi
    echo "Ласт: $DUMP_FILE"
elif [[ "$DUMP_FILE" != /* ]]; then
    DUMP_FILE="${DEST_DIR}/${DUMP_FILE}"
fi

echo "- Restoring $NAME ($HOST:$PORT / $DB) ---"

set +e
PGPASSWORD="$PASS" pg_restore \
    -h "$HOST" \
    -p "$PORT" \
    -U "$USER" \
    -d "$DB" \
    --clean \
    --if-exists \
    --no-owner \
    "$DUMP_FILE"
RESTORE_STATUS=$?
set -e

if [ $RESTORE_STATUS -eq 0 ]; then
    echo "OK: База $NAME успешно восстановлена!"
elif [ $RESTORE_STATUS -eq 1 ]; then
    echo "WARNING: Восстановление $NAME завершено с wArNiNg."
else
    echo "ERROR: Сбой восстановления $NAME: $RESTORE_STATUS!"
    exit $RESTORE_STATUS
fi