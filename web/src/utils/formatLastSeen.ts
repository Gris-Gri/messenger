const MONTH_NAMES_GENITIVE = [
  'января',
  'февраля',
  'марта',
  'апреля',
  'мая',
  'июня',
  'июля',
  'августа',
  'сентября',
  'октября',
  'ноября',
  'декабря',
] as const

function isSameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  )
}

function pluralRu(n: number, one: string, few: string, many: string): string {
  const abs = Math.abs(n) % 100
  const last = abs % 10
  if (abs > 10 && abs < 20) {
    return many
  }
  if (last === 1) {
    return one
  }
  if (last >= 2 && last <= 4) {
    return few
  }
  return many
}

/**
 * Относительная «последняя активность» для MembersPanel.
 * `null`, если даты нет или она невалидна — UI ничего не показывает.
 */
export function formatLastSeen(
  iso: string | null | undefined,
  now: Date = new Date(),
): string | null {
  if (!iso) {
    return null
  }

  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return null
  }

  const diffMs = Math.max(0, now.getTime() - date.getTime())
  const diffMin = Math.floor(diffMs / 60_000)

  if (diffMin < 1) {
    return 'был(а) в сети только что'
  }
  if (diffMin < 60) {
    return `был(а) в сети ${diffMin} ${pluralRu(diffMin, 'минуту', 'минуты', 'минут')} назад`
  }

  const diffHours = Math.floor(diffMin / 60)
  if (diffHours < 24 && isSameDay(date, now)) {
    return `был(а) в сети ${diffHours} ${pluralRu(diffHours, 'час', 'часа', 'часов')} назад`
  }

  const yesterday = new Date(now)
  yesterday.setDate(now.getDate() - 1)
  if (isSameDay(date, yesterday)) {
    return 'был(а) в сети вчера'
  }

  const day = date.getDate()
  const month = MONTH_NAMES_GENITIVE[date.getMonth()]
  if (date.getFullYear() === now.getFullYear()) {
    return `был(а) в сети ${day} ${month}`
  }

  return `был(а) в сети ${day} ${month} ${date.getFullYear()}`
}
