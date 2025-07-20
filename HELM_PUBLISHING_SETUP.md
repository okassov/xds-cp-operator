# Helm Chart Publishing Setup

Этот документ описывает автоматическую публикацию Helm chart в GitHub Pages по семантическим тэгам.

## 🚀 Быстрая настройка

### 1. Включите GitHub Pages

1. Перейдите в **Settings** → **Pages** вашего GitHub репозитория
2. В разделе **Source** выберите **GitHub Actions**
3. Сохраните настройки

### 2. Создайте семантический тэг для публикации

```bash
# Убедитесь что все изменения в Chart готовы
git add .
git commit -m "feat: prepare chart for v1.0.0 release"
git push origin main

# Создайте семантический тэг (format: v<major>.<minor>.<patch>)
git tag v1.0.0
git push origin v1.0.0
```

### 3. Дождитесь выполнения workflow

- Откройте вкладку **Actions** в GitHub репозитории
- Дождитесь завершения workflow **"Publish Helm Chart"**
- Chart автоматически обновится с версией из тэга (без префикса "v")
- После успешного выполнения chart будет доступен по адресу:
  `https://okassov.github.io/xds-cp-operator/`

## 📦 Использование опубликованного chart

### Добавление Helm repository

```bash
helm repo add xds-cp-operator https://okassov.github.io/xds-cp-operator/
helm repo update
```

### Установка оператора

```bash
# Последняя стабильная версия
helm install xds-cp-operator xds-cp-operator/xds-cp-operator \
  --namespace xds-system \
  --create-namespace

# Конкретная версия
helm install xds-cp-operator xds-cp-operator/xds-cp-operator \
  --namespace xds-system \
  --create-namespace \
  --version 1.0.0

# С кастомными параметрами  
helm install xds-cp-operator xds-cp-operator/xds-cp-operator \
  --namespace xds-system \
  --create-namespace \
  --version 1.0.0 \
  --set image.tag=1.0.0 \
  --set xdsService.type=LoadBalancer

# Pre-release версия
helm install xds-cp-operator xds-cp-operator/xds-cp-operator \
  --namespace xds-system \
  --create-namespace \
  --version 2.0.0-beta.1 \
  --set image.tag=2.0.0-beta.1
```

### Просмотр доступных версий

```bash
# Все версии
helm search repo xds-cp-operator/xds-cp-operator --versions

# Только stable версии
helm search repo xds-cp-operator/xds-cp-operator --versions | grep -v -E "(alpha|beta|rc)"

# Информация о chart
helm show chart xds-cp-operator/xds-cp-operator --version 1.0.0
```

## 🔄 Автоматическое обновление

Chart автоматически публикуется **только** при создании семантических тэгов:

### Поддерживаемые форматы тэгов:
- `v1.0.0` → Chart version: `1.0.0`
- `v1.2.3-beta.1` → Chart version: `1.2.3-beta.1`
- `v2.0.0+build.123` → Chart version: `2.0.0+build.123`

### Процесс публикации:
1. **Validation** - проверка семантического версионирования
2. **Update Chart.yaml** - автоматическое обновление `version` и `appVersion`
3. **Package** - упаковка chart с новой версией
4. **Publish** - публикация в GitHub Pages
5. **Update README** - обновление документации с примерами новой версии

### Создание новой версии:
```bash
# Для патча (1.0.0 → 1.0.1)
git tag v1.0.1
git push origin v1.0.1

# Для минорной версии (1.0.1 → 1.1.0)  
git tag v1.1.0
git push origin v1.1.0

# Для мажорной версии (1.1.0 → 2.0.0)
git tag v2.0.0
git push origin v2.0.0

# Для pre-release (2.0.0 → 2.1.0-alpha.1)
git tag v2.1.0-alpha.1
git push origin v2.1.0-alpha.1
```

## 🌐 Дополнительные возможности

### Artifact Hub

После настройки chart автоматически появится в [Artifact Hub](https://artifacthub.io/) - централизованном каталоге Helm charts.

Найти можно будет по адресу:
`https://artifacthub.io/packages/helm/xds-cp-operator/xds-cp-operator`

### Проверка status

Проверить статус публикации:

```bash
# Проверить содержимое repository
curl -s https://okassov.github.io/xds-cp-operator/index.yaml

# Поиск chart
helm search repo xds-cp-operator/xds-cp-operator --versions
```

## 🛠️ Troubleshooting

### Chart не обновляется

1. Проверьте статус GitHub Actions в разделе **Actions**
2. Убедитесь, что GitHub Pages включены
3. Проверьте permissions в Settings → Actions → General

### 404 ошибка при доступе к repository

1. Убедитесь, что GitHub Pages включены и настроены на **GitHub Actions**
2. Проверьте, что workflow успешно выполнился
3. Подождите несколько минут - GitHub Pages может обновляться с задержкой

### Невозможно найти chart

```bash
# Проверить корректность URL
helm repo add xds-cp-operator https://okassov.github.io/xds-cp-operator/ --force-update
helm repo update
helm search repo xds-cp-operator
```

## 📚 Дополнительные ресурсы

- [GitHub Pages Documentation](https://docs.github.com/en/pages)
- [Helm Chart Repository Guide](https://helm.sh/docs/topics/chart_repository/)
- [Artifact Hub](https://artifacthub.io/)

## 🔧 Расширенная конфигурация

### Кастомный домен

Если хотите использовать собственный домен:

1. Добавьте файл `CNAME` в root директорию с вашим доменом
2. Настройте DNS записи для вашего домена
3. В GitHub Settings → Pages укажите custom domain

### Приватный Helm repository

Для приватных chart используйте:
- [ChartMuseum](https://chartmuseum.com/)
- [Harbor](https://goharbor.io/)
- [GitHub Packages](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry) 