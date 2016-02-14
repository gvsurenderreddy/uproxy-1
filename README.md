#uproxy

Простой каскадный прокси, написанный на Go.
Работает как единая точка входа для большого количества апстрим-прокси, умеет проверять их работоспособность и задержки.

Запускается urpoxy -c config.yaml

Формат конфига:
```
# список апстрим-прокси, по одной на строчку, разделение запятыми
proxyList: /etc/proxy.list
check:
	# url, который используется для проверки апстрим прокси
    url: http://dev.brandspotter.ru/proxy
	# строка, которую мы ожидаем получить
    string: FriendshipIsMagic
	# как часто проверять все прокси из листа
    interval: 20m
	# таймаут, по достижению которого помечаем прокси как неработоспособную
    timeout: 5s
bind: "0.0.0.0:8888"
# количество тредов для проверки прокси
workersCount: 20
# количество попыток сменить прокси и повторить запрос при ошибке
maxTry: 2
debug: false
```

По SIGUSR1 выдаёт в лог список рабочих прокси и некоторую статистику по ним.
По URL http://BIND/status выдаёт количество рабочих прокси в листе и количество запросов к ним.
Для debian-based систем есть скрипт сборки make-deb.sh

##TODO
 - https support
 - auth support
