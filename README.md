# Subtle Race

(основано на реальном баге из стандартной библиотеки Go)

## Описание

Необходимо реализовать интерфейс cron'a:

```go
type Cron interface {
	Run(ctx context.Context, action func(), next func() time.Duration)
}
```

Реализация должна вызывать `action`, ожидая перед каждым запуском `delay()` времени.
После отмены контекста не должно быть вызовов `action`.

Такой подход на практике очень часто используется, например:
* HTTP ретраи
* Service discovery
* Adaptive polling
* Keepalive / heartbeat
* Любые задачи с backoff

## Задание

Мейнтейнеры Go тоже пытались реализовать такой интерфейс. Сходу у них получилась следующая реализация:

```go
func (c *cronImpl) Run(ctx context.Context, action func(), next func() time.Duration) {
	var t *time.Timer

	t = time.AfterFunc(next(), func() {
		select {
		case <-ctx.Done():
			return
		default:
			action()
			t.Reset(next())
		}
	})

	<-ctx.Done()
}
```

Можете попробовать запустить [lite_test](./internal/cron/lite_test) и проверить корректность тестами. Более того,
с race detector'ом эти тесты тоже проходят.

Изначально попробуйте глазами найти проблему в коде.


> Обратите внимание, в какой горутине и в какой момент происходит вызов `Reset`
```go
t.Reset(next())
```

Далее запустите [hard_test](./internal/cron/hard_test) **без** race detector'a,
после чего попробуйте по выводу теста понять, в чём проблема.

После чего запустите [hard_test](./internal/cron/hard_test) с race detector'ом и финально осознайте проблему.

Исправьте текущую реализацию.

## Сдача
* Решение необходимо реализовать в файле [cron.go](./internal/cron/cron.go)
* Открыть pull request из ветки `hw` в ветку `main` **вашего репозитория**
* В описании PR заполнить количество часов, которые вы потратили на это задание
* Не стоит изменять файлы в директории [.github](.github)

## Особенности реализации
* Желательно не использовать `time.Sleep` в реализации

## Скрипты
Для запуска скриптов на курсе необходимо установить [go-task](https://taskfile.dev/docs/installation)

`go install github.com/go-task/task/v3/cmd/task@latest`

Перед выполнением задания не забудьте выполнить:

```bash 
task update
```

Запустить линтер:
```bash 
task lint
```

Запустить тесты:
```bash
task test
``` 

Обновить файлы задания
```bash
task update
```

Принудительно обновить файлы задания
```bash
task force-update
```

Скрипты работают на Windows, однако при разработке на этой операционной системе
рекомендуется использовать [WSL](https://learn.microsoft.com/en-us/windows/wsl/install)
