# Политика пяти индикаторов, контракт v1.2

RouteReading.indicators имеет максимум 5 элементов. Базовый порядок:

| Слот | Категория |
| --- | --- |
| 1 | weather |
| 2 | fleet |
| 3 | trend |
| 4 | peak; при активном event factor — event |
| 5 | calendar |

Event заменяет только peak indicator в кадрах/итогах, пересекающих effective_window, если event factor отличается от 1. Вне окна снова отображается peak. Weather/season factors аннотируют соответствующие weather/calendar slots. Partial overlap явно описывается текстом.

Это приоритет отображения: load_index и прочие числовые metrics остаются в frames/totals, maximum summary считается по frames. Frontend должен выбирать indicator по key, не предполагать постоянный key у слота 4.

Одновременное отображение шести категорий требует совместного обновления OpenAPI и frontend. Текущая поставка сохраняет лимит v1.2.
