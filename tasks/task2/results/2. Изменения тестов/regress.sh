#!/bin/bash
set -euo pipefail

echo "🏁 Регрессионный тест Hotelio (bookings в отдельной БД)"

pass() { echo "✅ $1"; }
fail() { echo "❌ $1"; exit 1; }

: "${DB_HOST:?DB_HOST is required}"
: "${DB_PORT:?DB_PORT is required}"
: "${DB_NAME:?DB_NAME is required}"
: "${DB_USER:?DB_USER is required}"
: "${DB_PASSWORD:?DB_PASSWORD is required}"

: "${DB_HOST2:?DB_HOST2 is required}"
: "${DB_PORT2:?DB_PORT2 is required}"

DB_NAME2="${DB_NAME2:-$DB_NAME}"
DB_USER2="${DB_USER2:-$DB_USER}"
DB_PASSWORD2="${DB_PASSWORD2:-$DB_PASSWORD}"

BASE="${API_URL:-http://localhost:8080}"

echo "🧪 Проверка подключения к БД1 (основная)..."
timeout 2 bash -c "</dev/tcp/${DB_HOST}/${DB_PORT}" \
  || { echo "❌ Не удалось подключиться к ${DB_HOST}:${DB_PORT}"; exit 1; }

echo "🧪 Проверка подключения к БД2 (bookings)..."
timeout 2 bash -c "</dev/tcp/${DB_HOST2}/${DB_PORT2}" \
  || { echo "❌ Не удалось подключиться к ${DB_HOST2}:${DB_PORT2}"; exit 1; }

echo "🧪 Загрузка фикстур в БД1 (всё, кроме bookings)..."
PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" "${DB_NAME}" < init-fixtures.sql
rm -f "$tmp_fixtures"
trap - EXIT

echo "🧪 Загрузка фикстур в БД2 (bookings)..."
PGPASSWORD="${DB_PASSWORD2}" psql -h "${DB_HOST2}" -p "${DB_PORT2}" -U "${DB_USER2}" "${DB_NAME2}" < init-fixtures2.sql

echo "🧪 Выполнение HTTP-тестов..."

echo ""
echo "Тесты пользователей..."
curl -sSf "${BASE}/api/users/test-user-1" | grep -q 'Alice' && pass "Получение test-user-1 по ID работает" || fail "Пользователь test-user-1 не найден"
curl -sSf "${BASE}/api/users/test-user-1/status" | grep -q 'ACTIVE' && pass "Статус test-user-1: ACTIVE" || fail "Неверный статус пользователя"
curl -sSf "${BASE}/api/users/test-user-1/blacklisted" | grep -q 'true' && pass "test-user-1 в блэклисте" || fail "Блэклист не работает"
curl -sSf "${BASE}/api/users/test-user-1/active" | grep -q 'true' && pass "test-user-1 активен" || fail "Активность не работает"
curl -sSf "${BASE}/api/users/test-user-1/authorized" | grep -q 'false' && pass "test-user-1 не авторизован (в блэклисте)" || fail "Авторизация работает неправильно"
curl -sSf "${BASE}/api/users/test-user-3/vip" | grep -q 'true' && pass "test-user-3 — VIP-пользователь" || fail "VIP-статус не работает"
curl -sSf "${BASE}/api/users/test-user-2/authorized" | grep -q 'true' && pass "test-user-2 авторизован" || fail "Авторизация (true) не работает"

echo ""
echo "Тесты отелей..."
curl -sSf "${BASE}/api/hotels/test-hotel-1" | grep -q 'Seoul' && pass "test-hotel-1 получен по ID" || fail "test-hotel-1 не найден"
curl -sSf "${BASE}/api/hotels/test-hotel-1/operational" | grep -q 'true' && pass "test-hotel-1 работает" || fail "test-hotel-1 не работает"
curl -sSf "${BASE}/api/hotels/test-hotel-3/operational" | grep -q 'false' && pass "test-hotel-3 не работает" || fail "Статус работы test-hotel-3 некорректен"
curl -sSf "${BASE}/api/hotels/test-hotel-2/fully-booked" | grep -q 'true' && pass "test-hotel-2 полностью забронирован" || fail "Статус fullyBooked test-hotel-2 неверен"
curl -sSf "${BASE}/api/hotels/by-city?city=Seoul" | grep -q 'Seoul' && pass "Поиск отелей в Сеуле работает" || fail "Поиск отелей в Сеуле не работает"
curl -sSf "${BASE}/api/hotels/top-rated?city=Seoul&limit=1" | grep -q 'Seoul' && pass "Топ-отели в Сеуле загружены" || fail "Топ-отели не найдены"

echo ""
echo "Тесты ревью..."
curl -sSf "${BASE}/api/reviews/hotel/test-hotel-1" | grep -q 'Amazing experience' \
  && pass "Отзывы test-hotel-1 найдены" || fail "Отзывы test-hotel-1 не найдены"
curl -sSf "${BASE}/api/reviews/hotel/test-hotel-1/trusted" | grep -q 'true' \
  && pass "test-hotel-1 признан надёжным" || fail "Надёжность test-hotel-1 не определена"
curl -sSf "${BASE}/api/reviews/hotel/test-hotel-3/trusted" | grep -q 'false' \
  && pass "test-hotel-3 НЕ признан надёжным (ожидаемо)" || fail "Надёжность test-hotel-3 некорректно определена"

echo ""
echo "Тесты промокодов..."
curl -sSf "${BASE}/api/promos/TESTCODE1" | grep -q 'TESTCODE1' && pass "Промокод TESTCODE1 найден" || fail "Промокод TESTCODE1 не найден"
curl -sSf "${BASE}/api/promos/TESTCODE-VIP/valid?isVipUser=true" | grep -q 'true' && pass "VIP-промо доступен VIP" || fail "VIP-промо НЕ доступен VIP"
curl -sSf "${BASE}/api/promos/TESTCODE-VIP/valid?isVipUser=false" | grep -q 'false' && pass "VIP-промо недоступен обычному" || fail "VIP-промо доступен обычному"
curl -sSf "${BASE}/api/promos/TESTCODE1/valid" | grep -q 'true' && pass "Обычный промо доступен" || fail "Обычный промо недоступен"
curl -sSf "${BASE}/api/promos/TESTCODE-OLD/valid" | grep -q 'false' && pass "Истекший промо недоступен" || fail "Истекший промо доступен"
curl -sSf -X POST "${BASE}/api/promos/validate?code=TESTCODE1&userId=test-user-2" | grep -q 'TESTCODE1' && pass "POST /validate промо прошёл" || fail "POST /validate не прошёл"

echo ""
echo "Тесты бронирования..."
curl -sSf "${BASE}/api/bookings" | grep -q 'test-user-2' && pass "Все бронирования получены" || fail "Бронирования не получены"
curl -sSf "${BASE}/api/bookings?userId=test-user-2" | grep -q 'test-user-2' && pass "Бронирования test-user-2 найдены" || fail "Нет бронирований test-user-2"
curl -sSf -X POST "${BASE}/api/bookings?userId=test-user-3&hotelId=test-hotel-1" | grep -q 'test-hotel-1' && pass "Бронирование прошло (без промо)" || fail "Бронирование (без промо) не прошло"
curl -sSf -X POST "${BASE}/api/bookings?userId=test-user-2&hotelId=test-hotel-1&promoCode=TESTCODE1" | grep -q 'TESTCODE1' && pass "Бронирование с промо прошло" || fail "Бронирование с промо не прошло"

code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE}/api/bookings?userId=test-user-0&hotelId=test-hotel-1")
if [[ "$code" == "500" ]]; then
  pass "Отклонено: неактивный пользователь"
else
  fail "Ошибка: сервер принял бронирование от неактивного пользователя (код $code)"
fi

curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE}/api/bookings?userId=test-user-2&hotelId=test-hotel-3" | grep -q '500' \
  && pass "Отклонено: недоверенный отель" \
  || fail "Ошибка: сервер принял бронирование от недоверенного отеля"

curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE}/api/bookings?userId=test-user-2&hotelId=test-hotel-2" | grep -q '500' \
  && pass "Отклонено: отель полностью забронирован" \
  || fail "Ошибка: сервер принял бронирование в полностью занятом отеле"

echo "✅ Все HTTP-тесты пройдены!"
