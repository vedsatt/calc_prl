# Распределенный вычислитель арифметических выражений

## О проекте
Данный проект представляет собой систему для параллельного вычисления арифметических выражений в распределенной среде.
Логика вычисления выполняется агентом. Он создает указанное количество воркеров, которые одновременно считают бинарные выражения из основного выражения.

## Структура проекта
```
cmd/
  ├── agent/
  │   └── cmd.go
  ├── orchestrator/
  │   └── cmd.go
internal/
  ├── agent/
  │   ├── agent.go
  │   ├── worker.go
  │   └── worker_test.go
  ├── config/
  │   └── config.go
  ├── orchestrator/
  │   ├── calculator.go
  │   ├── calculator_test.go
  │   ├── handlers.go
  │   └── orchestrator.go
pkg/
  ├── ast/
  │   ├── ast.go
  │   ├── ast_test.go
  │   ├── build.go
  │   ├── errors.go
  │   ├── errors_test.go
  │   ├── rpn.go
  │   ├── rpn_test.go
  │   ├── tokens.go
  │   ├── tokens_test.go
  │   └── vars.go
  ├── database/
  │   ├── database.go
  │   └── database_test.go
.gitignore
go.mod
README.md
```


## Установка и запуск проекта
### Установка
1. Клонируйте репозиторий
```sh
git clone https://github.com/vedsatt/calc_prl.git
cd ./calc_prl
```
Установите зависимости
```sh
go mod tidy
```
### Запуск
1. **Создайте файл `.env`** в корневой папке и укажите в нем параметры:
```sh
TIME_ADDITION_MS=<ЗНАЧЕНИЕ>
TIME_SUBTRACTION_MS=<ЗНАЧЕНИЕ>
TIME_MULTIPLICATIONS_MS=<ЗНАЧЕНИЕ>
TIME_DIVISIONS_MS=<ЗНАЧЕНИЕ>
COMPUTING_POWER=<ЗНАЧЕНИЕ>
```
**Примечение:** если не создать файл и/или не указать определенные значения, то программа установит неуказанные значения по умолчанию

2. **Запустите оркестратор:**
```sh
go run ./cmd/orchestrator/cmd.go
```
3. **Запустите агента:**
```sh
go run ./cmd/agent/cmd.go
```
## Документация
### Оркестратор
Когда пользователь отправляет выражение на сервер, программа строит по выражению AST дерево и возвращает пользователю id выражения (если найдена ошибка, то она возвращается пользователю вместо id). Выражение добавляется в базу со статусом "in proccess" и начинается расчет.

Расчет происходит следующим образом: программа строит хеш-таблицу по AST дереву, для дальнейшей работы. Далее программа проходится по дереву, находя узлы, которые могут быть посчитаны и отправляет их в канал tasks. Когда агент обращается к серверу, в ответ он получает значение из канала tasks. Когда агент отдает результат значения, сервер отправляет его в канал results. Программа берет результат из этого канала, меняет значение посчитанного узла и удаляет его ветки (то есть, узлы с операндами). Это делается в хеш-таблице, поскольку ее ключи - айди узлов, а значения - указатели на узлы дерева. Таким образом, меняя значение в хэш-таблице, мы меняем значение и в дереве.После этого программа заново обходит дерево и повторяет предыдущие действия, пока окончательный результат не будет посчитан. Когда в мапе (хеш-таблице) остается 1 значение - это сигнализирует о том, что все дерево было посчитано. В таком случае программа отправляет конечный результат в канал last_result и обновляет значение в базе данных.

### Агент
При старте агент запускает несколько горутин, каждая из которых с коротким промежутком отправляет GET запрос на сервер. Если результат получен - горутина считает выражение, формирует ответ из ошибки и результата и отправляет его на сервер методом POST. Если выражение посчитано корректно - поле ошибки будет пустым, а результат будет иметь значение и наоборот.

## Компоненты системы
### Оркестратор
Оркестратор отвечает за прием арифметических выражений, разбиение их на отдельные операции и распределение задач агентам.

#### API Оркестратора
- **Добавление вычисления арифметического выражения**
```sh
curl --location 'localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "expression": "2+2*2"
}'
```
**Ответ:**
```json
{
    "id": <уникальный идентификатор выражения>
}
```
*или:*
```json
{
    "error": <ошибка>
}
```

- **Получение списка выражений**
```sh
curl --location 'localhost:8080/api/v1/expressions'
```
**Ответ:**
```json
{
    "expressions": [
        {
            "id": <идентификатор выражения>,
            "status": <статус вычисления выражения>,
            "result": <результат выражения>
        }
    ]
}
```
*или:*
```json
{
    "error": <ошибка>
}
```

- **Получение выражения по его идентификатору**
```sh
curl --location 'localhost:8080/api/v1/expressions/:id'
```
**Ответ:**
```json
{
    "expression": {
        "id": <идентификатор выражения>,
        "status": <статус вычисления выражения>,
        "result": <результат выражения>
    }
}
```
*или:*
```json
{
    "error": <ошибка>
}
```

- **Получение задачи для выполнения**
```sh
curl --location 'localhost:8080/internal/task'
```
**Ответ:**
```json
{
    "task": {
        "id": <идентификатор задачи>,
        "arg1": <имя первого аргумента>,
        "arg2": <имя второго аргумента>,
        "operation": <операция>
    }
}
```

- **Прием результата обработки данных**
```sh
curl --location 'localhost:8080/internal/task' \
--header 'Content-Type: application/json' \
--data '{
  "id": 1,
  "result": 2.5
}'
```

---

### Агент
Агент получает задачи от оркестратора, выполняет их и отправляет обратно результаты.
Агент запускает несколько вычислительных горутин, количество которых регулируется переменной `COMPUTING_POWER`.

## Примеры запросов и ответов для сервера

### 1. Добавление вычисления арифметического выражения

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "expression": "2 + 2 * 2"
}'
```

**Ответ**:
```json
{
    "id": 1741187030581847400
}
```

**Код ответа**:
- 201 - выражение принято для вычисления

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "expression": "2 / 0"
}'
```

**Ответ**:
```json
{
    "error": "division by zero"
}
```

**Код ответа**:
- 422 - невалидные данные

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "exp":
}'
```

**Ответ**:
```json
{
    "error": "internal server error"
}
```

**Код ответа**:
- 500 - некорректный запрос

---

### 2. Получение списка выражений

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/expressions'
```

**Ответ**:
```json
{
    "expressions": [
        {
            "id": 1741187753311405100,
            "status": "in process",
            "result": ""
        },
        {
            "id": 174118775331145300,
            "status": "done",
            "result": 6.0000
        }
    ]
}
```

**Код ответ**:
- 200 - успешно получен список выражений

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/expressions'
```

**Ответ**:
```json
{
    "error":"empty base"
}
```

**Код ответ**:
- 500 - база данных пустая

---

### 3. Получение выражения по его идентификатору

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/expressions/:1741187753311405100'
```

**Ответ**:
```json
{
    "expression": {
        "id": 1741187753311405100,
        "status": "in proccess",
        "result": ""
    }
}
```

**Код ответа**:
- 200 - успешно получено выражение

**Запрос**:
```bash
curl --location 'localhost:8080/api/v1/expressions/:1741187753311643100'
```

**Ответ**:
```json
{
    "error":"no expression with id: 1741187753311643100"
}
```

**Код ответа**:
- 404 - нет такого выражения
  
## Контакты
Если у вас возникли вопросы, предложения и т.д., вот мой тг:
```
https://t.me/sigmatemik52
```
