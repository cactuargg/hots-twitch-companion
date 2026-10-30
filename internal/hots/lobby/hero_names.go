// Hero names and attribute IDs extracted from HeroesToolChest/heroes-data, build 93640 (MIT).
// Source blob: b9592de561672ae3dca2f70de724b9f2fdb6eee5. See THIRD_PARTY_NOTICES.txt.
package lobby

import (
	"strings"
	"unicode"
)

type heroNames struct {
	en, ru  string
	aliases []string
}

var heroCatalog = map[string]heroNames{
	"Abat": {en: "Abathur", ru: "Абатур", aliases: []string{"Abathur", "Abathur"}},
	"Alar": {en: "Alarak", ru: "Аларак", aliases: []string{"Alarak", "Alarak"}},
	"Alex": {en: "Alexstrasza", ru: "Алекстраза", aliases: []string{"Alexstrasza", "Alexstrasza"}},
	"Amaz": {en: "Cassia", ru: "Кассия", aliases: []string{"Amazon", "Cassia"}},
	"HANA": {en: "Ana", ru: "Ана", aliases: []string{"Ana", "Ana"}},
	"Andu": {en: "Anduin", ru: "Андуин", aliases: []string{"Anduin", "Anduin"}},
	"Anub": {en: "Anub'arak", ru: "Ануб'арак", aliases: []string{"Anubarak", "Anubarak"}},
	"Arts": {en: "Artanis", ru: "Артанис", aliases: []string{"Artanis", "Artanis"}},
	"Arth": {en: "Arthas", ru: "Артас", aliases: []string{"Arthas", "Arthas"}},
	"Auri": {en: "Auriel", ru: "Ауриэль", aliases: []string{"Auriel", "Auriel"}},
	"Azmo": {en: "Azmodan", ru: "Азмодан", aliases: []string{"Azmodan", "Azmodan"}},
	"Barb": {en: "Sonya", ru: "Соня", aliases: []string{"Barbarian", "Sonya"}},
	"Butc": {en: "The Butcher", ru: "Мясник", aliases: []string{"Butcher", "TheButcher"}},
	"Chen": {en: "Chen", ru: "Чэнь", aliases: []string{"Chen", "Chen"}},
	"CCho": {en: "Cho", ru: "Чо", aliases: []string{"Cho", "Chogall"}},
	"Chro": {en: "Chromie", ru: "Хроми", aliases: []string{"Chromie", "Chromie"}},
	"Crus": {en: "Johanna", ru: "Джоанна", aliases: []string{"Crusader", "Johanna"}},
	"DVA0": {en: "D.Va", ru: "D.Va", aliases: []string{"DVa", "DVa"}},
	"DEAT": {en: "Deathwing", ru: "Смертокрыл", aliases: []string{"Deathwing", "Deathwing"}},
	"DECK": {en: "Deckard", ru: "Декард", aliases: []string{"Deckard", "Deckard"}},
	"Deha": {en: "Dehaka", ru: "Дехака", aliases: []string{"Dehaka", "Dehaka"}},
	"Demo": {en: "Valla", ru: "Валла", aliases: []string{"DemonHunter", "Valla"}},
	"Diab": {en: "Diablo", ru: "Диабло", aliases: []string{"Diablo", "Diablo"}},
	"Drya": {en: "Lunara", ru: "Лунара", aliases: []string{"Dryad", "Lunara"}},
	"Faer": {en: "Brightwing", ru: "Светик", aliases: []string{"FaerieDragon", "Brightwing"}},
	"Fals": {en: "Falstad", ru: "Фалстад", aliases: []string{"Falstad", "Falstad"}},
	"FENX": {en: "Fenix", ru: "Феникс", aliases: []string{"Fenix", "Fenix"}},
	"Fire": {en: "Blaze", ru: "Блэйз", aliases: []string{"Firebat", "Blaze"}},
	"Gall": {en: "Gall", ru: "Галл", aliases: []string{"Gall", "Gall"}},
	"Garr": {en: "Garrosh", ru: "Гаррош", aliases: []string{"Garrosh", "Garrosh"}},
	"Genj": {en: "Genji", ru: "Гэндзи", aliases: []string{"Genji", "Genji"}},
	"Genn": {en: "Greymane", ru: "Седогрив", aliases: []string{"Greymane", "Greymane"}},
	"Guld": {en: "Gul'dan", ru: "Гул'дан", aliases: []string{"Guldan", "Guldan"}},
	"Hanz": {en: "Hanzo", ru: "Хандзо", aliases: []string{"Hanzo", "Hanzo"}},
	"HOGG": {en: "Hogger", ru: "Дробитель", aliases: []string{"Hogger", "Hogger"}},
	"Illi": {en: "Illidan", ru: "Иллидан", aliases: []string{"Illidan", "Illidan"}},
	"IMPE": {en: "Imperius", ru: "Империй", aliases: []string{"Imperius", "Imperius"}},
	"Jain": {en: "Jaina", ru: "Джайна", aliases: []string{"Jaina", "Jaina"}},
	"Junk": {en: "Junkrat", ru: "Крысавчик", aliases: []string{"Junkrat", "Junkrat"}},
	"Kael": {en: "Kael'thas", ru: "Кель'тас", aliases: []string{"Kaelthas", "Kaelthas"}},
	"KelT": {en: "Kel'Thuzad", ru: "Кел'Тузад", aliases: []string{"KelThuzad", "KelThuzad"}},
	"Kerr": {en: "Kerrigan", ru: "Керриган", aliases: []string{"Kerrigan", "Kerrigan"}},
	"L90E": {en: "E.T.C.", ru: "E.T.C.", aliases: []string{"L90ETC", "ETC"}},
	"Leor": {en: "Leoric", ru: "Леорик", aliases: []string{"Leoric", "Leoric"}},
	"LiLi": {en: "Li Li", ru: "Ли Ли", aliases: []string{"LiLi", "LiLi"}},
	"Lost": {en: "The Lost Vikings", ru: "Потерявшиеся викинги", aliases: []string{"LostVikings", "LostVikings"}},
	"Luci": {en: "Lúcio", ru: "Лусио", aliases: []string{"Lucio", "Lucio"}},
	"Maie": {en: "Maiev", ru: "Майев", aliases: []string{"Maiev", "Maiev"}},
	"MalG": {en: "Mal'Ganis", ru: "Мал'Ганис", aliases: []string{"MalGanis", "MalGanis"}},
	"Malf": {en: "Malfurion", ru: "Малфурион", aliases: []string{"Malfurion", "Malfurion"}},
	"MALT": {en: "Malthael", ru: "Малтаэль", aliases: []string{"Malthael", "Malthael"}},
	"Medi": {en: "Lt. Morales", ru: "Лейтенант Моралес", aliases: []string{"Medic", "LtMorales"}},
	"Mdvh": {en: "Medivh", ru: "Медив", aliases: []string{"Medivh", "Medivh"}},
	"HMEI": {en: "Mei", ru: "Мэй", aliases: []string{"MeiOW", "Mei"}},
	"MEPH": {en: "Mephisto", ru: "Мефисто", aliases: []string{"Mephisto", "Mephisto"}},
	"Monk": {en: "Kharazim", ru: "Каразим", aliases: []string{"Monk", "Kharazim"}},
	"Mura": {en: "Muradin", ru: "Мурадин", aliases: []string{"Muradin", "Muradin"}},
	"Murk": {en: "Murky", ru: "Мурчаль", aliases: []string{"Murky", "Murky"}},
	"Necr": {en: "Xul", ru: "Зул", aliases: []string{"Necromancer", "Xul"}},
	"NXHU": {en: "Qhira", ru: "Кахира", aliases: []string{"NexusHunter", "Qhira"}},
	"Nova": {en: "Nova", ru: "Нова", aliases: []string{"Nova", "Nova"}},
	"ORPH": {en: "Orphea", ru: "Орфея", aliases: []string{"Orphea", "Orphea"}},
	"Prob": {en: "Probius", ru: "Пробиус", aliases: []string{"Probius", "Probius"}},
	"Ragn": {en: "Ragnaros", ru: "Рагнарос", aliases: []string{"Ragnaros", "Ragnaros"}},
	"Rayn": {en: "Raynor", ru: "Рейнор", aliases: []string{"Raynor", "Raynor"}},
	"Rehg": {en: "Rehgar", ru: "Регар", aliases: []string{"Rehgar", "Rehgar"}},
	"Rexx": {en: "Rexxar", ru: "Рексар", aliases: []string{"Rexxar", "Rexxar"}},
	"Samu": {en: "Samuro", ru: "Самуро", aliases: []string{"Samuro", "Samuro"}},
	"Sgth": {en: "Sgt. Hammer", ru: "Сержант Кувалда", aliases: []string{"SgtHammer", "SgtHammer"}},
	"Stit": {en: "Stitches", ru: "Стежок", aliases: []string{"Stitches", "Stitches"}},
	"STUK": {en: "Stukov", ru: "Стуков", aliases: []string{"Stukov", "Stukov"}},
	"Sylv": {en: "Sylvanas", ru: "Сильвана", aliases: []string{"Sylvanas", "Sylvanas"}},
	"Tass": {en: "Tassadar", ru: "Тассадар", aliases: []string{"Tassadar", "Tassadar"}},
	"Thra": {en: "Thrall", ru: "Тралл", aliases: []string{"Thrall", "Thrall"}},
	"Tink": {en: "Gazlowe", ru: "Газлоу", aliases: []string{"Tinker", "Gazlowe"}},
	"Tra0": {en: "Tracer", ru: "Трейсер", aliases: []string{"Tracer", "Tracer"}},
	"Tych": {en: "Tychus", ru: "Тайкус", aliases: []string{"Tychus", "Tychus"}},
	"Tyrl": {en: "Tyrael", ru: "Тираэль", aliases: []string{"Tyrael", "Tyrael"}},
	"Tyrd": {en: "Tyrande", ru: "Тиранда", aliases: []string{"Tyrande", "Tyrande"}},
	"Uthe": {en: "Uther", ru: "Утер", aliases: []string{"Uther", "Uther"}},
	"VALE": {en: "Valeera", ru: "Валира", aliases: []string{"Valeera", "Valeera"}},
	"Vari": {en: "Varian", ru: "Вариан", aliases: []string{"Varian", "Varian"}},
	"WHIT": {en: "Whitemane", ru: "Вайтмейн", aliases: []string{"Whitemane", "Whitemane"}},
	"Witc": {en: "Nazeebo", ru: "Назибо", aliases: []string{"WitchDoctor", "Nazeebo"}},
	"Wiza": {en: "Li-Ming", ru: "Ли-Мин", aliases: []string{"Wizard", "LiMing"}},
	"YREL": {en: "Yrel", ru: "Ирель", aliases: []string{"Yrel", "Yrel"}},
	"Zaga": {en: "Zagara", ru: "Загара", aliases: []string{"Zagara", "Zagara"}},
	"Zary": {en: "Zarya", ru: "Заря", aliases: []string{"Zarya", "Zarya"}},
	"Zera": {en: "Zeratul", ru: "Зератул", aliases: []string{"Zeratul", "Zeratul"}},
	"ZULJ": {en: "Zul'jin", ru: "Зул'джин", aliases: []string{"Zuljin", "Zuljin"}},
}

func heroText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}
func HeroName(code, language string) string {
	h, ok := heroCatalog[strings.TrimSpace(code)]
	if !ok {
		return ""
	}
	if language == "ru" {
		return h.ru
	}
	return h.en
}
func HeroMatches(code, name string) bool {
	h, ok := heroCatalog[strings.TrimSpace(code)]
	if !ok || name == "" {
		return false
	}
	n := heroText(name)
	if n == heroText(h.en) || n == heroText(h.ru) {
		return true
	}
	for _, a := range h.aliases {
		if n == heroText(a) {
			return true
		}
	}
	return false
}
