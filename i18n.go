package main

import "fmt"

// UI and backend message catalog. English is the canonical fallback;
// Russian preserves the original v0.1/v0.2 behaviour.
const (
	langEn = "en"
	langRu = "ru"
)

func normalizeLang(s string) string {
	if s == langRu {
		return langRu
	}
	return langEn
}

var messages = map[string]map[string]string{
	langEn: {
		"twitch.connect_first":        "Connect Twitch first",
		"twitch.token_expired":        "Token expired. Reconnect Twitch",
		"twitch.refresh_error":        "Token refresh error: %s",
		"twitch.incomplete_tokens":    "Twitch returned an incomplete token pair",
		"twitch.wrong_client":         "The token was issued for a different Client ID",
		"twitch.no_scope":             "Missing channel:manage:predictions permission",
		"twitch.prediction_exists":    "An unfinished Prediction already exists on Twitch",
		"twitch.create_bad_response":  "Unexpected Twitch response while creating the Prediction",
		"twitch.outcomes_invalid":     "Could not verify Twitch outcomes",
		"auth.code_expired":           "Twitch authorization code expired",
		"auth.no_tokens":              "Twitch did not return tokens",
		"auth.need_client_id":         "Set your Twitch Client ID in the Twitch tab and save settings first",
		"auth.requesting":             "Requesting Twitch code…",
		"auth.request_failed":         "Failed to request the code",
		"auth.enter_code":             "Enter the code and approve Predictions access",
		"config.stop_watch_first":     "Stop monitoring before changing settings",
		"config.mode_conflict_active": "An unfinished prediction from another mode exists. Finish it in the previous mode first",
		"config.auth_pending":         "Finish Twitch authorization before changing settings",
		"config.confirm_real":         "Confirm enabling real Predictions",
		"config.too_long":             "Setting value is too long",
		"config.lang_invalid":         "Language must be \"en\" or \"ru\"",
		"config.window_invalid":       "Prediction duration must be between 1 and 30 minutes",
		"automation.already_running":  "Already running",
		"automation.need_player":      "Set the HotS player name",
		"automation.need_client_id":   "Set the Twitch Client ID",
		"automation.mode_conflict":    "Prediction from another mode. Switch the mode back first",
		"settle.none_active":          "No active prediction",
		"lobby.not_found":             "battlelobby not found yet",
		"replay.player_matches":       "Player matched %d times — check the name or Toon Handle",
		"replay.unknown_outcome":      "Unknown match outcome",
		"replay.not_found":            "Replay not found. Check the folder in Settings.",
		"replay.read_error":           "Could not read %s: %s",
		"http.request_too_large":      "Request too large",
		"prediction.title":            "HotS win?",
		"prediction.win":              "Win",
		"prediction.loss":             "Loss",
		"mode.real":                   "REAL Predictions",
		"mode.simulation":             "simulation",
		"log.app_started":             "App started. Simulation mode protects real Twitch points.",
		"log.read_only":               "HotS files are opened read-only.",
		"log.twitch_connected":        "Twitch connected: @%s",
		"log.token_check":             "Checking saved Twitch token: %s",
		"log.auth_code_received":      "Twitch: authorization code received. Continue in the Twitch activation page.",
		"log.auth_error":              "Twitch authorization error: %s",
		"log.token_protect_error":     "Failed to protect the token: %s",
		"log.identity_failed":         "Token received, but Twitch validation failed: %s",
		"log.replay_no_player":        "Replay parsed, but the player was not identified: %s",
		"log.replay_check_error":      "Replay read error: %s",
		"log.replay_summary":          "Replay: %s · %s → %s",
		"log.config_saved":            "Settings saved",
		"log.automation_started":      "Automation started",
		"log.automation_stopping":     "Stopping monitoring",
		"log.shutdown_idle":           "Window closed or connection lost: shutting down",
		"log.http_server":             "Local server: %s",
		"log.sim_created":             "[TEST] Prediction created without calling the Twitch API",
		"log.prediction_created":      "Twitch: Prediction %s created",
		"log.sim_outcome":             "[TEST] Outcome: %s (no Twitch API)",
		"log.watch_started":           "HotS monitoring: %s",
		"log.new_game":                "New game: %s",
		"log.skip_lobby":              "Skipping lobby: prediction already running or cooldown",
		"log.timeout_cancel":          "Result unknown: refunding points via cancel",
		"log.cancel_error":            "Cancel error: %s",
		"log.result":                  "Result: %s = %s, %s",
		"log.settled":                 "Prediction settled: %s",
		"log.process_error":           "Could not process %s: %s",
		"log.create_error":            "Create error: %s",
		"log.watch_stopped":           "Monitoring stopped",
		"error.read_config":           "Error reading settings: %s",
		"error.startup":               "Startup error: %s",
		"error.open_ui":               "Could not open the interface: %s",
	},
	langRu: {
		"twitch.connect_first":        "Сначала подключи Twitch",
		"twitch.token_expired":        "Токен истёк. Подключи Twitch заново",
		"twitch.refresh_error":        "Ошибка обновления токена: %s",
		"twitch.incomplete_tokens":    "Twitch вернул неполную пару токенов",
		"twitch.wrong_client":         "Токен выдан другому Client ID",
		"twitch.no_scope":             "Нет разрешения channel:manage:predictions",
		"twitch.prediction_exists":    "На Twitch уже есть незавершённый Prediction",
		"twitch.create_bad_response":  "Непредвиденный ответ Twitch при создании Prediction",
		"twitch.outcomes_invalid":     "Не удалось проверить исходы Twitch",
		"auth.code_expired":           "Код авторизации Twitch истёк",
		"auth.no_tokens":              "Twitch не прислал токены",
		"auth.need_client_id":         "Сначала укажи Twitch Client ID в разделе Twitch и сохрани настройки",
		"auth.requesting":             "Запрашиваем код Twitch…",
		"auth.request_failed":         "Ошибка запроса кода",
		"auth.enter_code":             "Введи код и подтверди доступ к Predictions",
		"config.stop_watch_first":     "Останови наблюдение перед изменением настроек",
		"config.mode_conflict_active": "Есть незавершённый прогноз из другого режима. Сначала заверши его в прежнем режиме",
		"config.auth_pending":         "Заверши авторизацию Twitch перед изменением настроек",
		"config.confirm_real":         "Подтверди включение реальных Predictions",
		"config.too_long":             "Значение настройки слишком длинное",
		"config.lang_invalid":         "Язык должен быть «en» или «ru»",
		"config.window_invalid":       "Длительность Prediction должна быть от 1 до 30 минут",
		"automation.already_running":  "Уже запущено",
		"automation.need_player":      "Укажи ник игрока HotS",
		"automation.need_client_id":   "Укажи Twitch Client ID",
		"automation.mode_conflict":    "Прогноз из другого режима. Сначала переключи режим обратно",
		"settle.none_active":          "Нет активного прогноза",
		"lobby.not_found":             "battlelobby пока не найден",
		"replay.player_matches":       "Игрок найден %d раз — проверь имя или Toon Handle",
		"replay.unknown_outcome":      "Неизвестный исход матча",
		"replay.not_found":            "Реплей не найден. Проверь путь в настройках.",
		"replay.read_error":           "Не удалось прочитать %s: %s",
		"http.request_too_large":      "Слишком большой запрос",
		"prediction.title":            "Победа в HotS?",
		"prediction.win":              "Победа",
		"prediction.loss":             "Поражение",
		"mode.real":                   "РЕАЛЬНЫЕ Predictions",
		"mode.simulation":             "симуляция",
		"log.app_started":             "Приложение запущено. Режим симуляции защищает реальные баллы Twitch.",
		"log.read_only":               "Файлы HotS открываются только для чтения.",
		"log.twitch_connected":        "Twitch подключён: @%s",
		"log.token_check":             "Проверка сохранённого Twitch токена: %s",
		"log.auth_code_received":      "Twitch: код авторизации получен. Продолжи в браузере Twitch.",
		"log.auth_error":              "Ошибка авторизации Twitch: %s",
		"log.token_protect_error":     "Не удалось защитить токен: %s",
		"log.identity_failed":         "Токен получен, но Twitch не прошёл проверку: %s",
		"log.replay_no_player":        "Реплей прочитан, но игрок не определён: %s",
		"log.replay_check_error":      "Ошибка чтения реплея: %s",
		"log.replay_summary":          "Реплей: %s · %s → %s",
		"log.config_saved":            "Настройки сохранены",
		"log.automation_started":      "Автоматизация запущена",
		"log.automation_stopping":     "Мониторинг останавливается",
		"log.shutdown_idle":           "Окно закрыто или соединение потеряно: остановка приложения",
		"log.http_server":             "Локальный сервер: %s",
		"log.sim_created":             "[ТЕСТ] Prediction создан без обращения к Twitch API",
		"log.prediction_created":      "Twitch: создан Prediction %s",
		"log.sim_outcome":             "[ТЕСТ] Исход: %s (без Twitch API)",
		"log.watch_started":           "Мониторинг HotS: %s",
		"log.new_game":                "Новая игра: %s",
		"log.skip_lobby":              "Пропуск lobby: прогноз уже идёт либо cooldown",
		"log.timeout_cancel":          "Результат неизвестен: возврат баллов через отмену",
		"log.cancel_error":            "Ошибка отмены: %s",
		"log.result":                  "Результат: %s = %s, %s",
		"log.settled":                 "Prediction рассчитан: %s",
		"log.process_error":           "Не удалось обработать %s: %s",
		"log.create_error":            "Ошибка создания: %s",
		"log.watch_stopped":           "Наблюдение остановлено",
		"error.read_config":           "Ошибка чтения настроек: %s",
		"error.startup":               "Ошибка запуска: %s",
		"error.open_ui":               "Не удалось открыть интерфейс: %s",
	},
}

func tr(lang, key string, args ...any) string {
	table := messages[normalizeLang(lang)]
	s, ok := table[key]
	if !ok {
		if s, ok = messages[langEn][key]; !ok {
			return key
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}
