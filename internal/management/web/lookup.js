
(() => {
  const $ = id => document.getElementById(id), endpoint = location.pathname.replace(/\/lookup\/?$/, '/lookup/data'); let token = '', data = null, modelMetric = 'requests', modelRankMetric = 'value', tokenTrendGrain = 'hour', costTrendGrain = 'hour', recentPage = 1, recentPageSize = 25; const charts = {};
  const CPA_LOCALE_KEY='cli-proxy-language', CPA_THEME_KEY='cli-proxy-theme', LOOKUP_REMEMBER_KEY='credit-manager.lookup.token', LOCALES=['zh-CN','zh-TW','en','ru'], THEMES=['auto','white','light','dark'];
  const COPY = {
    '密钥自助查询': { 'zh-TW':'密鑰自助查詢', en:'Key self-service lookup', ru:'Самообслуживание ключа' },
    '密钥额度与使用统计': { 'zh-TW':'密鑰額度與使用統計', en:'Key quota and usage', ru:'Лимиты и использование ключа' },
    '语言': { 'zh-TW':'語言', en:'Language', ru:'Язык' }, '主题': { 'zh-TW':'主題', en:'Theme', ru:'Тема' },
    '跟随系统': { 'zh-TW':'跟隨系統', en:'System', ru:'Системная' }, '纯白': { 'zh-TW':'純白', en:'White', ru:'Белая' }, '羊毛纸': { 'zh-TW':'羊毛紙', en:'Parchment', ru:'Пергамент' }, '暗色': { 'zh-TW':'暗色', en:'Dark', ru:'Тёмная' },
    '语言与主题': { 'zh-TW':'語言與主題', en:'Language and theme', ru:'Язык и тема' },
    'Token 显示单位': { 'zh-TW':'Token 顯示單位', en:'Token display unit', ru:'Единица отображения токенов' },
    '费用显示货币': { 'zh-TW':'費用顯示貨幣', en:'Cost currency', ru:'Валюта стоимости' },
    '原始数量': { 'zh-TW':'原始數量', en:'Raw count', ru:'Исходное количество' },
    '千 (×1,000)': { 'zh-TW':'千 (×1,000)', en:'Thousand (×1,000)', ru:'Тысячи (×1,000)' },
    'k (×1,000)': { 'zh-TW':'k (×1,000)', en:'k (×1,000)', ru:'k (×1,000)' },
    '万 (×10,000)': { 'zh-TW':'萬 (×10,000)', en:'10 thousand (×10,000)', ru:'Десятки тысяч (×10,000)' },
    'w (×10,000)': { 'zh-TW':'w (×10,000)', en:'w (×10,000)', ru:'w (×10,000)' },
    '百万 (×1,000,000)': { 'zh-TW':'百萬 (×1,000,000)', en:'Million (×1,000,000)', ru:'Миллионы (×1,000,000)' },
    'm (×1,000,000)': { 'zh-TW':'m (×1,000,000)', en:'m (×1,000,000)', ru:'m (×1,000,000)' },
    '个': { 'zh-TW':'個', en:'个', ru:'个' }, '千': { 'zh-TW':'千', en:'千', ru:'千' }, 'k': { 'zh-TW':'k', en:'k', ru:'k' }, '万': { 'zh-TW':'萬', en:'万', ru:'万' }, 'w': { 'zh-TW':'w', en:'w', ru:'w' }, '百万': { 'zh-TW':'百萬', en:'百万', ru:'百万' }, 'm': { 'zh-TW':'m', en:'m', ru:'m' },
    '费用': { 'zh-TW':'費用', en:'Cost', ru:'Стоимость' },
    '输入密钥后仅可查看该密钥自身的数据': { 'zh-TW':'輸入密鑰後僅可查看該密鑰自身的資料', en:'After entering a key, only that key’s data is visible.', ru:'После ввода ключа видны только его данные.' },
    '粘贴 tk- 开头的密钥': { 'zh-TW':'貼上 tk- 開頭的密鑰', en:'Paste a tk- key', ru:'Вставьте ключ tk-' },
    '插件密钥': { 'zh-TW':'外掛密鑰', en:'Plugin key', ru:'Ключ плагина' },
    '查询': { 'zh-TW':'查詢', en:'Look up', ru:'Проверить' },
    '请选择': { 'zh-TW':'請選擇', en:'Select', ru:'Выберите' },
    '查询密钥额度': { 'zh-TW':'查詢密鑰額度', en:'Look up key quota', ru:'Проверить лимит ключа' },
    '粘贴 tk- 开头的密钥，查看额度、Token 限制与用量。': { 'zh-TW':'貼上 tk- 開頭的密鑰，查看額度、Token 限制與用量。', en:'Paste a tk- key to view quota, token limits, and usage.', ru:'Вставьте ключ tk-, чтобы увидеть лимит и использование.' },
    '记住密钥': { 'zh-TW':'記住密鑰', en:'Remember key', ru:'Запомнить ключ' },
    '更换密钥': { 'zh-TW':'更換密鑰', en:'Switch key', ru:'Сменить ключ' },
    '未勾选时仅用于本次查询，不会写入本机。勾选后保存在本机浏览器，方便下次自动填入。': { 'zh-TW':'未勾選時僅用於本次查詢，不會寫入本機。勾選後儲存在本機瀏覽器，方便下次自動填入。', en:'If unchecked, the key is used only for this lookup. If checked, it is stored in this browser for the next visit.', ru:'Без галочки ключ не сохраняется. С галочкой — только в этом браузере.' },
    '密钥仅用于本次请求，不会保存到页面、浏览器或地址栏。': { 'zh-TW':'密鑰僅用於本次請求，不會儲存到頁面、瀏覽器或網址列。', en:'The key is used only for this request and is not saved to the page, browser, or address bar.', ru:'Ключ используется только для этого запроса и не сохраняется на странице, в браузере или адресной строке.' },
    '密钥概览': { 'zh-TW':'密鑰概覽', en:'Key overview', ru:'Обзор ключа' },
    '额度使用': { 'zh-TW':'額度使用', en:'Quota usage', ru:'Использование лимита' },
    '含当前在途预占': { 'zh-TW':'含目前在途預占', en:'Includes current holds', ru:'Включая текущие резервы' },
    '使用统计': { 'zh-TW':'使用統計', en:'Usage analytics', ru:'Статистика использования' },
    '时间范围': { 'zh-TW':'時間範圍', en:'Time range', ru:'Период' },
    '今日': { 'zh-TW':'今天', en:'Today', ru:'Сегодня' }, '最近 7 天': { 'zh-TW':'最近 7 天', en:'Last 7 days', ru:'Последние 7 дней' }, '最近 30 天': { 'zh-TW':'最近 30 天', en:'Last 30 days', ru:'Последние 30 дней' }, '最近 90 天': { 'zh-TW':'最近 90 天', en:'Last 90 days', ru:'Последние 90 дней' }, '自定义范围': { 'zh-TW':'自訂範圍', en:'Custom range', ru:'Свой период' }, '全部时间': { 'zh-TW':'全部時間', en:'All time', ru:'За всё время' },
    '开始时间（UTC）': { 'zh-TW':'開始時間（UTC）', en:'Start time (UTC)', ru:'Начало (UTC)' }, '结束时间（UTC）': { 'zh-TW':'結束時間（UTC）', en:'End time (UTC)', ru:'Конец (UTC)' },
    '开始时间': { 'zh-TW':'開始時間', en:'Start time', ru:'Начало' }, '结束时间': { 'zh-TW':'結束時間', en:'End time', ru:'Конец' },
    '选择日期和时间': { 'zh-TW':'選擇日期和時間', en:'Select date and time', ru:'Выберите дату и время' },
    '上个月': { 'zh-TW':'上個月', en:'Previous month', ru:'Предыдущий месяц' }, '下个月': { 'zh-TW':'下個月', en:'Next month', ru:'Следующий месяц' },
    '减少小时': { 'zh-TW':'減少小時', en:'Decrease hours', ru:'Уменьшить часы' }, '减少分钟': { 'zh-TW':'減少分鐘', en:'Decrease minutes', ru:'Уменьшить минуты' },
    '增加小时': { 'zh-TW':'增加小時', en:'Increase hours', ru:'Увеличить часы' }, '增加分钟': { 'zh-TW':'增加分鐘', en:'Increase minutes', ru:'Увеличить минуты' },
    '清除': { 'zh-TW':'清除', en:'Clear', ru:'Очистить' }, '此刻': { 'zh-TW':'此刻', en:'Now', ru:'Сейчас' },
    '开始时间不能晚于结束时间': { 'zh-TW':'開始時間不能晚於結束時間', en:'Start time cannot be after end time', ru:'Время начала не может быть позже времени окончания' },
    '模型': { 'zh-TW':'模型', en:'Model', ru:'Модель' }, '全部模型': { 'zh-TW':'全部模型', en:'All models', ru:'Все модели' },
    '刷新统计': { 'zh-TW':'重新整理統計', en:'Refresh stats', ru:'Обновить статистику' },
    'Token 趋势': { 'zh-TW':'Token 趨勢', en:'Token trend', ru:'Динамика токенов' },
    '费用趋势': { 'zh-TW':'費用趨勢', en:'Cost trend', ru:'Динамика расходов' },
    '趋势时间维度': { 'zh-TW':'趨勢時間維度', en:'Trend interval', ru:'Интервал графика' },
    'Token 趋势时间维度': { 'zh-TW':'Token 趨勢時間維度', en:'Token trend interval', ru:'Интервал токенов' },
    '费用趋势时间维度': { 'zh-TW':'費用趨勢時間維度', en:'Cost trend interval', ru:'Интервал расходов' },
    '时': { 'zh-TW':'時', en:'Hour', ru:'Час' },
    '日': { 'zh-TW':'日', en:'Day', ru:'День' },
    '月': { 'zh-TW':'月', en:'Month', ru:'Месяц' },
    '个有使用记录的小时': { 'zh-TW':'個有使用紀錄的小時', en:' hours with usage', ru:' часов с данными' },
    '个有使用记录的自然日': { 'zh-TW':'個有使用紀錄的自然日', en:' days with usage', ru:' дней с данными' },
    '个有使用记录的自然月': { 'zh-TW':'個有使用紀錄的自然月', en:' months with usage', ru:' месяцев с данными' },
    '模型调用占比': { 'zh-TW':'模型呼叫占比', en:'Model usage share', ru:'Доля вызовов моделей' },
    '模型效率排行': { 'zh-TW':'模型效率排行', en:'Model efficiency rank', ru:'Рейтинг эффективности' },
    '模型效率指标': { 'zh-TW':'模型效率指標', en:'Efficiency metric', ru:'Метрика эффективности' },
    '性价比': { 'zh-TW':'性價比', en:'Value', ru:'Выгода' },
    '单次': { 'zh-TW':'單次', en:'Per call', ru:'За запрос' },
    '吞吐': { 'zh-TW':'吞吐', en:'Throughput', ru:'Скорость' },
    '按每美元产出 Token 排序，越高越省': { 'zh-TW':'依每美元產出 Token 排序，越高越省', en:'Ranked by tokens per dollar; higher is cheaper', ru:'По токенам на доллар; больше — выгоднее' },
    '按每人民币产出 Token 排序，越高越省': { 'zh-TW':'依每人民幣產出 Token 排序，越高越省', en:'Ranked by tokens per yuan; higher is cheaper', ru:'По токенам на юань; больше — выгоднее' },
    '平均每次请求费用，越低越省': { 'zh-TW':'平均每次請求費用，越低越省', en:'Average cost per request; lower is cheaper', ru:'Средняя цена запроса; меньше — выгоднее' },
    '缓存读取占输入比例，越高越省': { 'zh-TW':'快取讀取佔輸入比例，越高越省', en:'Cache-read share of input; higher is cheaper', ru:'Доля чтения кэша во входе; больше — выгоднее' },
    '平均生成速度，越高越快': { 'zh-TW':'平均生成速度，越高越快', en:'Average generation speed; higher is faster', ru:'Средняя скорость генерации; больше — быстрее' },
    '当前筛选条件下暂无模型效率数据': { 'zh-TW':'目前篩選條件下暫無模型效率資料', en:'No model efficiency data for the current filters', ru:'Нет данных об эффективности моделей' },
    '免费': { 'zh-TW':'免費', en:'Free', ru:'Бесплатно' },
    '次': { 'zh-TW':'次', en:'calls', ru:'вызов.' },
    '调用次数': { 'zh-TW':'呼叫次數', en:'Requests', ru:'Запросы' },
    '输入': { 'zh-TW':'輸入', en:'Input', ru:'Вход' }, '输出': { 'zh-TW':'輸出', en:'Output', ru:'Выход' }, '缓存读取': { 'zh-TW':'快取讀取', en:'Cache read', ru:'Чтение кэша' }, '缓存命中率': { 'zh-TW':'快取命中率', en:'Cache hit rate', ru:'Попадания в кэш' },
    'Token 趋势图': { 'zh-TW':'Token 趨勢圖', en:'Token trend chart', ru:'График токенов' },
    '费用趋势图': { 'zh-TW':'費用趨勢圖', en:'Cost trend chart', ru:'График расходов' },
    '模型调用占比图': { 'zh-TW':'模型呼叫占比圖', en:'Model share chart', ru:'График доли моделей' },
    '最近调用': { 'zh-TW':'最近呼叫', en:'Recent calls', ru:'Последние вызовы' },
    '请选择自定义范围的开始和结束日期': { 'zh-TW':'請選擇自訂範圍的開始和結束日期', en:'Choose the custom start and end dates', ru:'Выберите начальную и конечную даты' },
    '开始日期不能晚于结束日期': { 'zh-TW':'開始日期不能晚於結束日期', en:'Start date cannot be after end date', ru:'Дата начала не может быть позже даты окончания' },
    '不限制': { 'zh-TW':'不限制', en:'Unlimited', ru:'Без ограничений' },
    '已用': { 'zh-TW':'已用', en:'Used', ru:'Использовано' },
    '剩余': { 'zh-TW':'剩餘', en:'Remaining', ru:'Осталось' },
    '最大并发': { 'zh-TW':'最大併發', en:'Max concurrency', ru:'Макс. параллельность' },
    '当前在途': { 'zh-TW':'目前在途', en:'In flight', ru:'В пути' },
    '个请求': { 'zh-TW':'個請求', en:'requests', ru:'запросов' },
    '当前在途请求数': { 'zh-TW':'目前在途請求數', en:'Current in-flight requests', ru:'Текущие незавершённые запросы' },
    '当前筛选条件下暂无数据': { 'zh-TW':'目前篩選條件下暫無資料', en:'No data for the current filters', ru:'Нет данных для текущих фильтров' },
    '图表组件加载失败': { 'zh-TW':'圖表元件載入失敗', en:'Chart component failed to load', ru:'Не удалось загрузить график' },
    '请求次数': { 'zh-TW':'請求次數', en:'Requests', ru:'Запросы' },
    '当前筛选条件下暂无模型使用数据': { 'zh-TW':'目前篩選條件下暫無模型使用資料', en:'No model usage for the current filters', ru:'Нет данных по моделям для текущих фильтров' },
    '当前筛选条件下暂无 Token 趋势': { 'zh-TW':'目前篩選條件下暫無 Token 趨勢', en:'No token trend for the current filters', ru:'Нет динамики токенов для текущих фильтров' },
    '当前筛选条件下暂无费用趋势': { 'zh-TW':'目前篩選條件下暫無費用趨勢', en:'No cost trend for the current filters', ru:'Нет динамики расходов для текущих фильтров' },
    '显示': { 'zh-TW':'顯示', en:'Showing', ru:'Показано' },
    '共': { 'zh-TW':'共', en:'of', ru:'из' },
    '条': { 'zh-TW':'條', en:' items', ru:' записей' },
    '每页': { 'zh-TW':'每頁', en:'Per page', ru:'На странице' },
    '上一页': { 'zh-TW':'上一頁', en:'Previous', ru:'Назад' },
    '下一页': { 'zh-TW':'下一頁', en:'Next', ru:'Далее' },
    '第': { 'zh-TW':'第', en:'Page', ru:'Стр.' },
    '页': { 'zh-TW':'頁', en:'', ru:'' },
    '我的密钥': { 'zh-TW':'我的密鑰', en:'My key', ru:'Мой ключ' },
    '指纹': { 'zh-TW':'指紋', en:'Fingerprint', ru:'Отпечаток' },
    '总额度': { 'zh-TW':'總額度', en:'Total quota', ru:'Общий лимит' },
    '日额度': { 'zh-TW':'日額度', en:'Daily quota', ru:'Дневной лимит' },
    '周额度': { 'zh-TW':'週額度', en:'Weekly quota', ru:'Недельный лимит' },
    '月额度': { 'zh-TW':'月額度', en:'Monthly quota', ru:'Месячный лимит' },
    '模型 Token 限制': { 'zh-TW':'模型 Token 限制', en:'Model token limits', ru:'Лимиты токенов модели' },
    '日 Token': { 'zh-TW':'日 Token', en:'Daily tokens', ru:'Токены за день' },
    '周 Token': { 'zh-TW':'週 Token', en:'Weekly tokens', ru:'Токены за неделю' },
    '月 Token': { 'zh-TW':'月 Token', en:'Monthly tokens', ru:'Токены за месяц' },
    '可用': { 'zh-TW':'可用', en:'Available', ru:'Доступно' },
    '禁用': { 'zh-TW':'停用', en:'Disabled', ru:'Отключено' },
    '未匹配模型': { 'zh-TW':'未匹配模型', en:'Unmatched models', ru:'Несовпавшие модели' },
    '无限制': { 'zh-TW':'無限制', en:'Unlimited', ru:'Без лимита' },
    '未匹配模型可用，未列入的模型不限制 Token。': { 'zh-TW':'未匹配模型可用，未列入的模型不限制 Token。', en:'Unmatched models are allowed with no token cap.', ru:'Несовпавшие модели доступны без лимита токенов.' },
    '未匹配模型已禁用，仅列出的模型可调用。': { 'zh-TW':'未匹配模型已停用，僅列出的模型可呼叫。', en:'Unmatched models are blocked; only listed models can be called.', ru:'Несовпавшие модели запрещены; можно вызывать только из списка.' },
    '累计已结算与在途预占': { 'zh-TW':'累計已結算與在途預占', en:'Settled spend plus current holds', ru:'Расчёт плюс текущие резервы' },
    'UTC 自然日': { 'zh-TW':'UTC 自然日', en:'UTC calendar day', ru:'Календарный день UTC' },
    'UTC 周一开始': { 'zh-TW':'UTC 週一開始', en:'UTC week starting Monday', ru:'Неделя UTC с понедельника' },
    'UTC 自然月': { 'zh-TW':'UTC 自然月', en:'UTC calendar month', ru:'Календарный месяц UTC' },
    '请求数': { 'zh-TW':'請求數', en:'Requests', ru:'Запросы' },
    '请求': { 'zh-TW':'請求', en:'requests', ru:'запросов' },
    '在途请求': { 'zh-TW':'在途請求', en:'In-flight requests', ru:'Незавершённые запросы' },
    '时间': { 'zh-TW':'時間', en:'Time', ru:'Время' }, '模型名称': { 'zh-TW':'模型名稱', en:'Model name', ru:'Модель' }, '来源': { 'zh-TW':'來源', en:'Source', ru:'Источник' }, '结果': { 'zh-TW':'結果', en:'Result', ru:'Результат' },
    '首字延迟': { 'zh-TW':'首字延遲', en:'First-token latency', ru:'Задержка первого токена' }, '生成时间': { 'zh-TW':'生成時間', en:'Generation time', ru:'Время генерации' },
    '思考强度': { 'zh-TW':'思考強度', en:'Thinking intensity', ru:'Интенсивность мышления' },
    '输入 Token': { 'zh-TW':'輸入 Token', en:'Input tokens', ru:'Входные токены' }, '输出 Token': { 'zh-TW':'輸出 Token', en:'Output tokens', ru:'Выходные токены' },
    '思考 Token': { 'zh-TW':'思考 Token', en:'Reasoning tokens', ru:'Токены рассуждения' },
    '缓存读取 Token': { 'zh-TW':'快取讀取 Token', en:'Cache-read tokens', ru:'Токены чтения кэша' },
    '缓存创建 Token': { 'zh-TW':'快取建立 Token', en:'Cache-creation tokens', ru:'Токены создания кэша' },
    '总 Token 数': { 'zh-TW':'總 Token 數', en:'Total tokens', ru:'Всего токенов' },
    '缓存命中': { 'zh-TW':'快取命中', en:'Cache hit', ru:'Попадание в кэш' },
    '是': { 'zh-TW':'是', en:'Yes', ru:'Да' }, '否': { 'zh-TW':'否', en:'No', ru:'Нет' },
    '请输入密钥。': { 'zh-TW':'請輸入密鑰。', en:'Enter a key.', ru:'Введите ключ.' },
    '密钥包含非英文字符或不可见字符，请重新复制完整的 tk- 开头密钥。': { 'zh-TW':'密鑰包含非英文字元或不可見字元，請重新複製完整的 tk- 開頭密鑰。', en:'The key contains non-ASCII or invisible characters. Copy the full tk- key again.', ru:'Ключ содержит не-ASCII или невидимые символы. Скопируйте полный ключ tk- ещё раз.' },
    '密钥格式不正确，请输入 tk- 开头的完整密钥。': { 'zh-TW':'密鑰格式不正確，請輸入 tk- 開頭的完整密鑰。', en:'Invalid key format. Enter the full tk- key.', ru:'Неверный формат ключа. Введите полный ключ tk-.' },
    '加载中…': { 'zh-TW':'載入中…', en:'Loading…', ru:'Загрузка…' },
    '查询失败，请检查密钥。': { 'zh-TW':'查詢失敗，請檢查密鑰。', en:'Lookup failed. Check the key.', ru:'Запрос не удался. Проверьте ключ.' },
    '查询失败，请稍后重试。': { 'zh-TW':'查詢失敗，請稍後重試。', en:'Lookup failed. Try again later.', ru:'Запрос не удался. Повторите позже.' },
    '入': { 'zh-TW':'入', en:'In', ru:'Вх.' }, '出': { 'zh-TW':'出', en:'Out', ru:'Исх.' }, '缓存': { 'zh-TW':'快取', en:'Cache', ru:'Кэш' }, '命中': { 'zh-TW':'命中', en:'Hit', ru:'Попад.' },
    '占比': { 'zh-TW':'佔比', en:'Share', ru:'Доля' },
    '美元兑人民币汇率': { 'zh-TW':'美元兌人民幣匯率', en:'USD to CNY rate', ru:'Курс USD к CNY' },
    '实时美元兑人民币汇率': { 'zh-TW':'即時美元兌人民幣匯率', en:'Live USD to CNY rate', ru:'Курс USD к CNY' },
    '汇率获取失败，已使用上次汇率': { 'zh-TW':'匯率取得失敗，已使用上次匯率', en:'Could not refresh the rate; using the last saved value', ru:'Не удалось обновить курс; используется прошлое значение' },
  };
  const textSources = new WeakMap(), attributeSources = new WeakMap();
  let translationObserver, locale='zh-CN', theme='auto';
  function parseStoredPreference(value) { if (!value) return ''; try { const parsed=JSON.parse(value); return parsed && typeof parsed==='object' ? (parsed.state||parsed).language || (parsed.state||parsed).theme || '' : parsed; } catch(_) { return value; } }
  function localeFromCPA() { const stored=String(parseStoredPreference(localStorage.getItem(CPA_LOCALE_KEY))||''); if (LOCALES.includes(stored)) return stored; const raw=String((navigator.languages&&navigator.languages[0])||navigator.language||'zh-CN').replace('_','-').toLowerCase(); if (['zh-tw','zh-hk','zh-mo','zh-hant'].some(prefix=>raw.startsWith(prefix))) return 'zh-TW'; if (raw.startsWith('zh')) return 'zh-CN'; if (raw.startsWith('ru')) return 'ru'; return 'en'; }
  function themeFromCPA() { const value=String(parseStoredPreference(localStorage.getItem(CPA_THEME_KEY))||'auto').toLowerCase(); return THEMES.includes(value)?value:'auto'; }
  function persistCPA(key, field, value) { try { const payload={state:{},version:0}; payload.state[field]=value; localStorage.setItem(key, JSON.stringify(payload)); } catch(_) {} }
  function buildCopyReverse() { const reverse=Object.create(null); Object.keys(COPY).forEach(key=>{ const row=COPY[key]; if(!row) return; ['zh-TW','en','ru'].forEach(loc=>{ const val=row[loc]; if(!val||val===key) return; if(!(val in reverse)) reverse[val]=key; else if(reverse[val]!==key) reverse[val]=null; }); }); return reverse; }
  let COPY_REVERSE=null;
  function copyReverse() { if(!COPY_REVERSE) COPY_REVERSE=buildCopyReverse(); return COPY_REVERSE; }
  function canonicalSource(text) { const trimmed=String(text==null?'':text).trim(); if(!trimmed) return trimmed; if(COPY[trimmed]) return trimmed; const mapped=copyReverse()[trimmed]; return mapped||trimmed; }
  function t(source) { const key=canonicalSource(source); if(locale==='zh-CN') return key; return (COPY[key] && COPY[key][locale]) || key; }
  function translateElement(element) { if (!element || element.closest('[data-no-i18n]')) return; ['title','placeholder','aria-label'].forEach(name=>{ if (!element.hasAttribute(name)) return; let values=attributeSources.get(element); if (!values) { values={}; attributeSources.set(element, values); } if (!(name in values)) values[name]=element.getAttribute(name); values[name]=canonicalSource(values[name]); element.setAttribute(name, t(values[name])); }); const textNodes=[...element.childNodes].filter(node=>node.nodeType===Node.TEXT_NODE && node.nodeValue.trim()); if (!textNodes.length) return; let values=textSources.get(element); if (!values) { values=new Map(); textSources.set(element, values); } textNodes.forEach(node=>{ if (!values.has(node)) values.set(node, node.nodeValue); const stored=values.get(node), leading=stored.match(/^\s*/)[0], trailing=stored.match(/\s*$/)[0], key=canonicalSource(stored.trim()); values.set(node, leading+key+trailing); node.nodeValue=leading+t(key)+trailing; }); }
  function translateTree(root) { if (!root) return; if (root.nodeType===Node.ELEMENT_NODE) translateElement(root); if (root.querySelectorAll) root.querySelectorAll('*').forEach(translateElement); }
  const TOKEN_UNITS={raw:{div:1,suffix:'',maxFrac:0},qian:{div:1e3,suffix:'千',maxFrac:2},k:{div:1e3,suffix:'k',maxFrac:2},wan:{div:1e4,suffix:'万',maxFrac:2},w:{div:1e4,suffix:'w',maxFrac:2},baiwan:{div:1e6,suffix:'百万',maxFrac:2},m:{div:1e6,suffix:'m',maxFrac:2}}, TOKEN_UNIT_KEY='credit-manager.token-unit', CURRENCY_KEY='credit-manager.currency', USD_CNY_RATE_KEY='credit-manager.usd-cny-rate', DEFAULT_USD_CNY_RATE=7.2;
  let tokenUnit=TOKEN_UNITS[localStorage.getItem(TOKEN_UNIT_KEY)]?localStorage.getItem(TOKEN_UNIT_KEY):'raw', currency=localStorage.getItem(CURRENCY_KEY)==='CNY'?'CNY':'USD', usdCnyRate=Math.max(0.0001,Number(localStorage.getItem(USD_CNY_RATE_KEY))||DEFAULT_USD_CNY_RATE);
  const count = value => { const n=Number(value); return Number.isFinite(n)?Math.round(n).toLocaleString():'—'; }, tokens = value => { const n=Number(value); if(!Number.isFinite(n))return '—'; const unit=TOKEN_UNITS[tokenUnit], scaled=n/unit.div, abs=Math.abs(scaled), maxFrac=abs>0&&abs<.01?6:(abs>=100?1:unit.maxFrac); return scaled.toLocaleString(undefined,{maximumFractionDigits:maxFrac})+unit.suffix; }, tokenValue = value => tokens(value)+' Token', money = value => { const usd=Number(value||0)/1e6; if(currency==='CNY') { const abs=Math.abs(usd*usdCnyRate), maxFrac=abs>=100?2:(abs>=1?3:4); return '¥'+(usd*usdCnyRate).toLocaleString(undefined,{maximumFractionDigits:maxFrac}); } return '$'+usd.toLocaleString(undefined,{maximumFractionDigits:6}); }, usd = value => money(value)+' '+currency, esc = v => String(v==null?'':v).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  function syncDisplayControls() { document.querySelectorAll('#tokenUnitSwitch [data-token-unit]').forEach(button=>button.classList.toggle('active',button.dataset.tokenUnit===tokenUnit)); document.querySelectorAll('#currencySwitch [data-currency]').forEach(button=>button.classList.toggle('active',button.dataset.currency===currency)); $('usdCnyRateField').classList.toggle('hidden',currency!=='CNY'); if($('usdCnyRate')) $('usdCnyRate').textContent=formatUsdCnyRate(usdCnyRate); }
   function formatUsdCnyRate(value) { const n=Math.max(0.0001,Number(value)||DEFAULT_USD_CNY_RATE); return n.toLocaleString(undefined,{minimumFractionDigits:2,maximumFractionDigits:2}); }
   function fxURL(refresh) { return location.pathname.replace(/\/lookup\/?$/,'/fx/usd-cny')+(refresh?'?refresh=1':''); }
   async function fetchUsdCnyRate(refresh) { const field=$('usdCnyRateField'), valueEl=$('usdCnyRate'); try { const response=await fetch(fxURL(refresh),{cache:'no-store',headers:{Accept:'application/json'}}), result=await response.json(), next=Number(result&&result.usd_to_cny); if(!response.ok||!Number.isFinite(next)||next<=0) throw new Error((result&&result.error)||'汇率获取失败'); usdCnyRate=next; localStorage.setItem(USD_CNY_RATE_KEY,String(next)); if(valueEl) valueEl.textContent=formatUsdCnyRate(next); if(field){ const asOf=result.fetched_at?new Date(result.fetched_at).toLocaleString():''; field.title=t('实时美元兑人民币汇率')+(asOf?' · '+asOf:''); } refreshDisplayUnits(); } catch(_) { if(valueEl) valueEl.textContent=formatUsdCnyRate(usdCnyRate); if(field) field.title=t('汇率获取失败，已使用上次汇率'); } }
  function refreshDisplayUnits() { if(data) { render(data); renderRecentPagination(data.recent_pagination); } }
  const totalTokens = item => Number(item.input_tokens||0)+Number(item.output_tokens||0);
  function syncCustomRange() { $('customRange').classList.toggle('hidden',$('range').value!=='custom'); }
  let customRangeLoadTimer;
  function queueCustomRangeLoad() {
    if ($('range').value!=='custom' || !$('from').value || !$('to').value) return;
    clearTimeout(customRangeLoadTimer);
    customRangeLoadTimer = setTimeout(()=>load(false,true), 280);
  }
  function rangeQuery() { const value = $('range').value; if (value === 'all') return ''; if (value === 'custom') { const from=$('from').value, to=$('to').value; if (!from || !to) throw new Error('请选择自定义范围的开始和结束日期'); const fromDate=new Date(from), toDate=new Date(to); if (Number.isNaN(fromDate.getTime()) || Number.isNaN(toDate.getTime())) throw new Error('请选择自定义范围的开始和结束日期'); if (fromDate>toDate) throw new Error('开始时间不能晚于结束时间'); return 'from='+encodeURIComponent(fromDate.toISOString())+'&to='+encodeURIComponent(toDate.toISOString()); } const now = new Date(), from = new Date(now); if (value === 'today') from.setUTCHours(0,0,0,0); else from.setUTCDate(now.getUTCDate()-Number(value)+1); from.setUTCHours(0,0,0,0); return 'from='+encodeURIComponent(from.toISOString())+'&to='+encodeURIComponent(now.toISOString()); }
  function uiIcon(name) {
    const s = 'viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"';
    const icons = {
      key: '<svg '+s+'><circle cx="5.6" cy="10.4" r="3.1"/><path d="m7.9 8.1 5.2-5.2M10.7 5.3l1.9 1.9M12.5 3.5l1.7 1.7"/></svg>',
      search: '<svg '+s+'><circle cx="7" cy="7" r="4.25"/><path d="m10.2 10.2 3 3"/></svg>',
      lock: '<svg '+s+'><rect x="3.2" y="7" width="9.6" height="6.5" rx="1.8"/><path d="M5.4 7V5.2a2.6 2.6 0 0 1 5.2 0V7M8 9.9v1.4"/></svg>',
      wallet: '<svg '+s+' stroke-width="1.5"><path d="M2.5 5.5a1.8 1.8 0 0 1 1.8-1.8h7.4a1.8 1.8 0 0 1 1.8 1.8v6a1.8 1.8 0 0 1-1.8 1.8H4.3a1.8 1.8 0 0 1-1.8-1.8z"/><path d="M9.5 8.5h4M2.5 5.5h9.2"/></svg>',
      cpu: '<svg '+s+'><rect x="4" y="4" width="8" height="8" rx="1.6"/><rect x="6.4" y="6.4" width="3.2" height="3.2" rx=".8"/><path d="M6.5 4V2.2M9.5 4V2.2M6.5 13.8V12M9.5 13.8V12M4 6.5H2.2M4 9.5H2.2M13.8 6.5H12M13.8 9.5H12"/></svg>',
      chart: '<svg '+s+'><path d="M2.5 13.5h11"/><path d="M4.2 13.2V8.4M7.6 13.2V4.2M11 13.2V6.4"/></svg>',
      coin: '<svg '+s+' stroke-width="1.5"><circle cx="8" cy="8" r="5.4"/><path d="M10.2 6.2c-.4-.7-1.2-1.1-2.2-1.1-1.3 0-2.3.7-2.3 1.6 0 2.2 4.6.9 4.6 2.7 0 .9-1 1.6-2.3 1.6-1 0-1.9-.4-2.3-1.1M8 3.9v8.2"/></svg>',
      pie: '<svg '+s+'><path d="M8 2.4a5.6 5.6 0 1 1-5.6 5.6H8z"/><path d="M10 2.6A5.6 5.6 0 0 1 13.4 6H10z"/></svg>',
      trophy: '<svg '+s+'><path d="M5 2.5h6v3.2a3 3 0 0 1-6 0z"/><path d="M5 3.4H3a2 2 0 0 0 2 3M11 3.4h2a2 2 0 0 1-2 3M8 8.7v2.1M5.6 13.5h4.8M6.8 13.3c0-1.4.5-2.5 1.2-2.5s1.2 1.1 1.2 2.5"/></svg>',
      list: '<svg '+s+'><path d="M5.4 4h8M5.4 8h8M5.4 12h8"/><circle cx="2.7" cy="4" r=".9" fill="currentColor" stroke="none"/><circle cx="2.7" cy="8" r=".9" fill="currentColor" stroke="none"/><circle cx="2.7" cy="12" r=".9" fill="currentColor" stroke="none"/></svg>',
      layers: '<svg '+s+'><path d="m8 2 5.6 2.9L8 7.8 2.4 4.9z"/><path d="m2.4 8 5.6 2.9L13.6 8M2.4 11.1l5.6 2.9 5.6-2.9"/></svg>',
      chevronL: '<svg '+s+'><path d="M10 3.5 5.5 8l4.5 4.5"/></svg>',
      chevronR: '<svg '+s+'><path d="m6 3.5 4.5 4.5L6 12.5"/></svg>',
      activity: '<svg '+s+'><path d="M1.8 8.2h2.7l1.6-4.4 2.8 8 1.7-3.6h3.6"/></svg>',
      trend: '<svg '+s+'><path d="m2.5 11 3.5-3.5 2.4 2.4 4.6-5"/><path d="M9.8 4.9h3.2v3.2"/></svg>',
      bolt: '<svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path d="M9.2 1.5 3.6 9h3.2l-.9 5.5L11.9 7H8.6z"/></svg>'
    };
    return icons[name] || '';
  }
  function quota(name, used, maximum, detail, wide=false) { const card='quota-card'+(wide?' wide':''); if (!Number(maximum||0)) return '<div class="'+card+'"><div class="quota-top"><span class="quota-name">'+esc(t(name))+'</span><span class="unlimited">'+esc(t('不限制'))+'</span></div><div class="quota-used">'+money(used)+'</div><div class="quota-detail">'+esc(t('已用'))+' · '+esc(t(detail))+'</div></div>'; const percent = Math.min(100,Math.max(0,Number(used)/Number(maximum)*100)), tone = percent>=90?'danger':percent>=70?'warn':''; return '<div class="'+card+'"><div class="quota-top"><span class="quota-name">'+esc(t(name))+'</span><span class="quota-cap">'+percent.toFixed(0)+'%</span></div><div class="quota-used">'+money(used)+'<small>/ '+money(maximum)+'</small></div><div class="quota-detail">'+esc(t('剩余'))+' '+money(Math.max(0,Number(maximum)-Number(used)))+'</div><div class="progress '+tone+'"><i style="width:'+percent+'%"></i></div></div>'; }
  function tokenPeriodBar(label, used, period) {
    const cap = Number(period && period.tokens || 0);
    const usedN = Number(used||0);
    if (!(cap > 0)) {
      const mode = (period && period.mode) === 'available' ? '可用' : '不限制';
      return '<div class="token-period"><div class="token-period-top"><span>'+esc(t(label))+'</span><b>'+tokenValue(usedN)+'</b><em>'+esc(t(mode))+'</em></div></div>';
    }
    const percent = Math.min(100, Math.max(0, usedN / cap * 100));
    const tone = percent>=90?'danger':percent>=70?'warn':'';
    return '<div class="token-period"><div class="token-period-top"><span>'+esc(t(label))+'</span><b>'+tokenValue(usedN)+'<small>/ '+tokenValue(cap)+'</small></b><strong class="quota-cap">'+percent.toFixed(0)+'%</strong></div><div class="progress '+tone+'"><i style="width:'+percent.toFixed(1)+'%"></i></div></div>';
  }
  function renderModelTokenLimits(result) {
    const usage = result.model_token_usage || [];
    const limits = (result.key && result.key.model_token_limits) || [];
    const unmatched = (result.key && result.key.unmatched_models_mode) === 'disabled';
    const section = $('modelTokenLimitsSection');
    const items = usage.length ? usage : limits.map(limit => ({ model:limit.model, total:limit.total, daily:limit.daily, weekly:limit.weekly, monthly:limit.monthly, total_used:0, daily_used:0, weekly_used:0, monthly_used:0 }));
    if (!items.length && !unmatched) { section.classList.add('hidden'); $('modelTokenLimits').innerHTML = ''; return; }
    section.classList.remove('hidden');
    $('modelTokenLimitsState').textContent = unmatched ? t('未匹配模型已禁用，仅列出的模型可调用。') : t('未匹配模型可用，未列入的模型不限制 Token。');
    const cards = items.map(item => '<div class="quota-card token-limit-card"><div class="quota-top"><span class="quota-name" title="'+esc(item.model)+'">'+esc(item.model)+'</span></div><div class="token-period-grid">'+tokenPeriodBar('总 Token', item.total_used, item.total)+tokenPeriodBar('日 Token', item.daily_used, item.daily)+tokenPeriodBar('周 Token', item.weekly_used, item.weekly)+tokenPeriodBar('月 Token', item.monthly_used, item.monthly)+'</div></div>');
    if (!cards.length) cards.push('<div class="quota-card token-limit-card"><div class="quota-top"><span class="quota-name">'+esc(t('未匹配模型'))+'</span><span class="unlimited">'+esc(unmatched ? t('禁用') : t('可用'))+'</span></div></div>');
    $('modelTokenLimits').innerHTML = cards.join('');
  }
  function concurrent(active, maximum) { if (!Number(maximum||0)) return '<div class="quota-card"><div class="quota-top"><span class="quota-name">'+esc(t('最大并发'))+'</span><span class="unlimited">'+esc(t('不限制'))+'</span></div><div class="quota-used">'+count(active)+'</div><div class="quota-detail">'+esc(t('当前在途'))+'</div></div>'; const p=Math.min(100,Number(active)/Number(maximum)*100), tone=p>=90?'danger':p>=70?'warn':''; return '<div class="quota-card"><div class="quota-top"><span class="quota-name">'+esc(t('最大并发'))+'</span><span class="quota-cap">'+Math.round(p)+'%</span></div><div class="quota-used">'+count(active)+'<small>/ '+count(maximum)+'</small></div><div class="quota-detail">'+esc(t('当前在途'))+'</div><div class="progress '+tone+'"><i style="width:'+p+'%"></i></div></div>'; }
  function table(headers, rows) { const head='<table><thead><tr>'+headers.map(h=>'<th>'+esc(t(h))+'</th>').join('')+'</tr></thead>'; return rows.length?head+'<tbody>'+rows.join('')+'</tbody></table>':head+'<tbody><tr class="empty-row"><td colspan="'+headers.length+'"><div class="empty"><span class="empty-icon" aria-hidden="true">'+uiIcon('list')+'</span><span>'+esc(t('当前筛选条件下暂无数据'))+'</span></div></td></tr></tbody></table>'; }
  function duration(value) { if(value==null||value==='')return '—'; const n=Number(value);return Number.isFinite(n)?(n<1000?(n<100?n.toFixed(1):n.toFixed(0))+' ms':(n/1000).toFixed(2)+' s'):'—'; }
  function tps(value) { const n=Number(value); return Number.isFinite(n)?n.toFixed(2):'—'; }
  function allTokens(item) { const reported=Number(item.total_tokens||0); if(reported>0) return reported; const total=Number(item.input_tokens||0)+Number(item.output_tokens||0)+Number(item.reasoning_tokens||0); return total>0?total:Number(item.cached_tokens||0); }
  const palette=['#7968ee','#1eb787','#f0a04b','#e56b8a','#4dabf7','#20c997','#ff922b','#cc5de8'];
  function destroyChart(id) { if(charts[id]) charts[id].dispose(); delete charts[id]; }
  function chart(id, empty, option, icon) { const target=$(id); destroyChart(id); if(!option||!window.echarts) { target.className='chart chart-empty'; target.innerHTML='<span class="empty-icon" aria-hidden="true">'+uiIcon(icon||'chart')+'</span><span>'+(window.echarts?esc(empty):esc(t('图表组件加载失败')))+'</span>'; return; } target.className='chart'; target.textContent=''; const instance=echarts.init(target,null,{renderer:'svg'}); instance.setOption(option); charts[id]=instance; requestAnimationFrame(()=>requestAnimationFrame(()=>instance.resize())); }
  function getTrendGrain(chart) { const grain=chart==='cost'?costTrendGrain:tokenTrendGrain; return grain==='hour'||grain==='month'?grain:'day'; }
  function defaultTrendGrain(range) { return range==='today'?'hour':(range==='90'||range==='all'?'month':'day'); }
  function trendLabel(dateKey, grain) { const value=String(dateKey||''); if(grain==='hour'){ const parts=value.split('T'); return (parts[0]||'').slice(5)+' '+(parts[1]||'00').slice(0,2)+':00'; } return grain==='month'?(value.slice(0,7)||value):(value.length>=10?value.slice(5,10):value); }
  function bucketTrend(points, grain) { const buckets=new Map(); (points||[]).forEach(point=>{ const raw=String(point.date||''); const key=grain==='month'?raw.slice(0,7):(grain==='day'?raw.slice(0,10):raw); if(!key) return; const current=buckets.get(key)||{date:key,input_tokens:0,output_tokens:0,cached_tokens:0,cache_read_tokens:0,cache_creation_tokens:0,cost_micro_usd:0}; current.input_tokens+=Number(point.input_tokens||0); current.output_tokens+=Number(point.output_tokens||0); current.cached_tokens+=Number(point.cached_tokens||0); current.cache_read_tokens+=Number(point.cache_read_tokens||0); current.cache_creation_tokens+=Number(point.cache_creation_tokens||0); current.cost_micro_usd+=Number(point.cost_micro_usd||0); buckets.set(key,current); }); return [...buckets.entries()].sort((a,b)=>a[0].localeCompare(b[0])).map(([,point])=>({...point,date:trendLabel(point.date,grain)})); }
  function syncTrendGrainSwitch(chart) { (chart?[chart]:['token','cost']).forEach(name=>{ const grain=getTrendGrain(name); document.querySelectorAll('.trend-grain-switch[data-trend-chart="'+name+'"] [data-trend-grain]').forEach(btn=>btn.classList.toggle('active',btn.dataset.trendGrain===grain)); }); }
  function setTrendGrain(chart, grain) { const next=grain==='hour'||grain==='month'?grain:'day'; if(chart==='cost') costTrendGrain=next; else tokenTrendGrain=next; syncTrendGrainSwitch(chart); if(data) renderCharts(data); }
  function trendChip(label, value, color, series, dashed) { return '<button type="button" class="trend-metric'+(dashed?' dashed':'')+'" data-series="'+esc(series)+'"><i style="background:'+color+'"></i>'+esc(t(label))+' <b>'+esc(value)+'</b></button>'; }
  function bindTrendChips(rowID, chartID) { const row=$(rowID); if(!row||!charts[chartID]) return; row.querySelectorAll('[data-series]').forEach(chip=>chip.addEventListener('click',()=>{ const instance=charts[chartID]; if(!instance) return; instance.dispatchAction({type:'legendToggleSelect',name:chip.dataset.series}); chip.classList.toggle('off'); })); }
  function lineChart(id, points, series, empty, moneyAxis=false) { if(!points.length) return chart(id,empty,null,moneyAxis?'coin':'chart'); const right=series.some(x=>x.right), sparse=points.length<=6, zoom=points.length>10; chart(id,empty,{color:series.map(x=>x.color),animationDuration:360,grid:{top:12,right:16,bottom:zoom?36:8,left:8,containLabel:true},legend:{show:false,data:series.map(x=>x.name)},tooltip:{trigger:'axis',backgroundColor:'rgba(26,31,28,.94)',borderWidth:0,padding:[9,11],textStyle:{color:'#fff',fontSize:12},axisPointer:{type:'line',lineStyle:{color:'#9ea9a1',type:'dashed'}},formatter:params=>'<strong>'+esc(params[0].axisValue)+'</strong><br/>'+params.map(param=>param.marker+series[param.seriesIndex].name+' <b style="float:right;margin-left:18px">'+series[param.seriesIndex].format(param.value)+'</b>').join('<br/>')},xAxis:{type:'category',boundaryGap:false,data:points.map(x=>x.date),axisLine:{lineStyle:{color:'#dce5de'}},axisTick:{show:false},axisLabel:{color:'#6a766e',fontSize:11,hideOverlap:true,margin:10}},yAxis:[{type:'value',min:0,splitNumber:3,axisLabel:{color:'#6a766e',fontSize:11,formatter:v=>moneyAxis?money(v):tokens(v)},splitLine:{lineStyle:{color:'#edf1ee',type:'dashed'}},axisLine:{show:false},axisTick:{show:false}},...(right?[{type:'value',min:0,max:100,splitNumber:2,axisLabel:{color:'#c56b82',fontSize:11,formatter:v=>v.toFixed(0)+'%'},splitLine:{show:false},axisLine:{show:false},axisTick:{show:false}}]:[])],dataZoom:zoom?[{type:'inside',start:0,end:100,zoomOnMouseWheel:'shift',moveOnMouseMove:true},{type:'slider',height:14,bottom:2,start:0,end:100,borderColor:'transparent',backgroundColor:'#f0f4f1',fillerColor:'rgba(30,183,135,.20)',handleStyle:{color:'#1eb787',borderColor:'#1eb787'},textStyle:{color:'transparent'}}]:[],series:series.map(s=>({name:s.name,type:'line',smooth:sparse?false:0.35,showSymbol:points.length<=24,symbol:'circle',symbolSize:s.dashed?4:6,yAxisIndex:s.right?1:0,lineStyle:{width:s.dashed?1.6:2.4,type:s.dashed?'dashed':'solid',opacity:s.dashed?.8:1},itemStyle:{borderWidth:2,borderColor:'#fff'},areaStyle:s.fill?{color:s.fill}:undefined,emphasis:{focus:'series',scale:true},data:points.map(x=>Number(x[s.key]||0))}))}); }
  function shareMetricValue(item) { if(modelMetric==='tokens')return Number(item.total_tokens||0); if(modelMetric==='cost')return Number(item.cost_micro_usd||0); return Number(item.request_count||0); }
  function shareMetricLabel() { return modelMetric==='tokens'?'Token':modelMetric==='cost'?t('费用'):t('请求次数'); }
  function shareMetricFormat(value) { return modelMetric==='tokens'?tokenValue(value):modelMetric==='cost'?usd(value):count(value)+' '+t('请求'); }
  function renderModelShare(models) { const sorted=[...models].sort((a,b)=>shareMetricValue(b)-shareMetricValue(a)), total=sorted.reduce((sum,x)=>sum+shareMetricValue(x),0), legend=$('modelShareLegend'); document.querySelectorAll('#modelMetric [data-metric]').forEach(button=>button.classList.toggle('active',button.dataset.metric===modelMetric)); if(!sorted.length||!total) { legend.innerHTML=''; return chart('modelShare',t('当前筛选条件下暂无模型使用数据'),null,'pie'); } legend.innerHTML=sorted.map((item,index)=>'<div class="share-legend-item" title="'+esc(item.model)+'"><i class="share-legend-dot" style="background:'+palette[index%palette.length]+'"></i><span class="share-legend-name">'+esc(item.model)+'</span><span class="share-legend-value">'+(shareMetricValue(item)/total*100).toFixed(1)+'%</span></div>').join(''); chart('modelShare',t('当前筛选条件下暂无模型使用数据'),{color:sorted.map((_,i)=>palette[i%palette.length]),animationDuration:360,tooltip:{trigger:'item',backgroundColor:'rgba(26,31,28,.94)',borderWidth:0,padding:[9,11],textStyle:{color:'#fff',fontSize:12},formatter:param=>'<strong>'+esc(param.data.item.model)+'</strong><br/>'+esc(shareMetricLabel())+' <b style="float:right;margin-left:18px">'+esc(shareMetricFormat(param.value))+'</b><br/>'+esc(t('占比'))+' <b style="float:right;margin-left:18px">'+param.percent.toFixed(1)+'%</b>'},series:[{name:shareMetricLabel(),type:'pie',radius:['48%','72%'],center:['50%','52%'],minAngle:3,avoidLabelOverlap:true,itemStyle:{borderColor:'#fff',borderWidth:3,borderRadius:4},label:{show:true,position:'center',formatter:'{total|'+shareMetricFormat(total)+'}\n{caption|'+shareMetricLabel()+'}',rich:{total:{color:'#1f2922',fontSize:18,fontWeight:700,lineHeight:24},caption:{color:'#5f6a61',fontSize:11,fontWeight:600,lineHeight:16}}},labelLine:{show:false},emphasis:{scale:true,scaleSize:7},data:sorted.map(item=>({name:item.model,value:shareMetricValue(item),item}))}]}); }
  function displaySpend(micro) { const usdVal=Number(micro||0)/1e6; return currency==='CNY'?usdVal*usdCnyRate:usdVal; }
  function lookupModelUsage(models) { return (models||[]).map(item=>{ const tokenCount=Number(item.total_tokens||0)||(Number(item.input_tokens||0)+Number(item.output_tokens||0)); const tpsVal=Number(item.avg_tokens_per_second||0); return { model:item.model||'未标记模型', requests:Number(item.request_count||0), tokens:tokenCount, cost:Number(item.cost_micro_usd||0), cacheRead:Number(item.cache_read_tokens||0), cacheRelated:Number(item.input_tokens||0)||Number(item.cache_read_tokens||0), tpsSum:tpsVal, tpsCount:tpsVal>0?1:0 }; }); }
  function modelRankSpec(metric) { const key=metric||((modelRankMetric==='unit'||modelRankMetric==='cache'||modelRankMetric==='tps')?modelRankMetric:'value'); if(key==='unit') return { key, label:'单次', hint:'平均每次请求费用，越低越省', higher:false, ready:item=>item.requests>0, score:item=>item.requests>0?item.cost/item.requests:NaN, format:value=>money(value) }; if(key==='cache') return { key, label:'缓存', hint:'缓存读取占输入比例，越高越省', higher:true, ready:item=>item.cacheRelated>0, score:item=>item.cacheRelated>0?item.cacheRead/item.cacheRelated*100:NaN, format:value=>Number(value).toFixed(1)+'%' }; if(key==='tps') return { key, label:'吞吐', hint:'平均生成速度，越高越快', higher:true, ready:item=>item.tpsCount>0, score:item=>item.tpsCount>0?item.tpsSum/item.tpsCount:NaN, format:value=>Number(value).toFixed(value>=10?1:2)+' tok/s' }; return { key:'value', label:'性价比', hint:currency==='CNY'?'按每人民币产出 Token 排序，越高越省':'按每美元产出 Token 排序，越高越省', higher:true, ready:item=>item.tokens>0, score:item=>{ const spend=displaySpend(item.cost); if(spend<=0) return item.tokens>0?Infinity:0; return item.tokens/spend; }, format:value=>!Number.isFinite(value)?t('免费'):(tokens(value)+(currency==='CNY'?'/¥':'/$')) }; }
  function rankBarPercent(item, ranked, spec) { if(!spec.ready(item)) return 0; const score=spec.score(item); if(score===Infinity) return 100; const finite=ranked.filter(spec.ready).map(spec.score).filter(Number.isFinite); if(!finite.length||!Number.isFinite(score)) return 0; if(spec.higher) { const max=Math.max(...finite); return max>0?Math.max(8,score/max*100):0; } const min=Math.min(...finite.filter(value=>value>0)); if(!Number.isFinite(min)||score<=0) return score<=0?100:0; return Math.max(8,min/score*100); }
  function drillLookupModel(model) { const name=String(model||'').trim(); if(!name) return; const select=$('model'); if(![...select.options].some(opt=>opt.value===name)) { const option=document.createElement('option'); option.value=name; option.textContent=name; select.appendChild(option); } select.value=name; load(false,true); }
  function renderModelRank(models) { const spec=modelRankSpec(); const ranked=lookupModelUsage(models).sort((a,b)=>{ const aReady=spec.ready(a), bReady=spec.ready(b); if(aReady!==bReady) return aReady?-1:1; if(!aReady) return a.model.localeCompare(b.model); const aScore=spec.score(a), bScore=spec.score(b); if(aScore===bScore) return b.requests-a.requests||a.model.localeCompare(b.model); if(aScore===Infinity) return -1; if(bScore===Infinity) return 1; if(!Number.isFinite(aScore)) return 1; if(!Number.isFinite(bScore)) return -1; return spec.higher?(bScore-aScore):(aScore-bScore); }); const target=$('modelRank'), hint=$('modelRankHint'); document.querySelectorAll('#modelRankMetric [data-rank-metric]').forEach(button=>button.classList.toggle('active',button.dataset.rankMetric===spec.key)); if(hint) hint.textContent=t(spec.hint); if(!target) return; if(!ranked.length) { target.innerHTML='<div class="chart-empty"><span class="empty-icon" aria-hidden="true">'+uiIcon('trophy')+'</span><span>'+esc(t('当前筛选条件下暂无模型效率数据'))+'</span></div>'; return; } target.innerHTML=ranked.map((item,index)=>{ const ready=spec.ready(item), score=ready?spec.score(item):NaN, value=ready&&(Number.isFinite(score)||score===Infinity)?spec.format(score):'—', width=rankBarPercent(item,ranked,spec), title=['value','unit','cache','tps'].map(key=>{ const metric=modelRankSpec(key), metricScore=metric.ready(item)?metric.score(item):NaN, metricValue=metric.ready(item)&&(Number.isFinite(metricScore)||metricScore===Infinity)?metric.format(metricScore):'—'; return t(metric.label)+' '+metricValue; }).join('\n'); return '<button type="button" class="model-rank-item" data-model="'+esc(item.model)+'" title="'+esc(title)+'"><span class="model-rank-index">'+(index+1)+'</span><span class="model-rank-body"><span class="model-rank-top"><span class="model-rank-name">'+esc(item.model)+'</span><span class="model-rank-value">'+esc(value)+'</span></span><span class="model-rank-bar" aria-hidden="true"><i style="width:'+width.toFixed(1)+'%"></i></span><span class="model-rank-meta">'+esc(count(item.requests)+' '+t('次')+' · '+tokenValue(item.tokens)+' · '+money(item.cost))+'</span></span></button>'; }).join(''); target.querySelectorAll('.model-rank-item').forEach(btn=>btn.addEventListener('click',()=>drillLookupModel(btn.dataset.model))); }
  function withHitRate(points) { return points.map(point=>{ const related=Number(point.cached_tokens||0)+Number(point.cache_read_tokens||0)+Number(point.cache_creation_tokens||0); return {...point, cache_hit_rate:related>0?Number(point.cache_read_tokens||0)/related*100:0}; }); }
  function renderCharts(result) { const raw=result.daily_trend||[], tokenPoints=withHitRate(bucketTrend(raw,getTrendGrain('token'))), costPoints=bucketTrend(raw,getTrendGrain('cost')), models=result.by_model||[], input=tokenPoints.reduce((sum,p)=>sum+Number(p.input_tokens||0),0),output=tokenPoints.reduce((sum,p)=>sum+Number(p.output_tokens||0),0),cache=tokenPoints.reduce((sum,p)=>sum+Number(p.cache_read_tokens||0),0),related=tokenPoints.reduce((sum,p)=>sum+Number(p.cached_tokens||0)+Number(p.cache_read_tokens||0)+Number(p.cache_creation_tokens||0),0), cost=costPoints.reduce((sum,p)=>sum+Number(p.cost_micro_usd||0),0); syncTrendGrainSwitch(); $('tokenTrendTotal').innerHTML=tokenPoints.length?trendChip('入',tokens(input),'#1eb787','输入')+trendChip('出',tokens(output),'#7968ee','输出')+trendChip('缓存',tokens(cache),'#f0a04b','缓存读取')+(related?trendChip('命中',(cache/related*100).toFixed(1)+'%','#e56b8a','缓存命中率',true):''):''; $('costTrendTotal').innerHTML=costPoints.length?trendChip('费用',money(cost),'#1eb787','费用'):''; lineChart('tokenTrend',tokenPoints,[{key:'input_tokens',name:t('输入'),color:'#1eb787',format:tokenValue},{key:'output_tokens',name:t('输出'),color:'#7968ee',format:tokenValue},{key:'cache_read_tokens',name:t('缓存读取'),color:'#f0a04b',format:tokenValue},{key:'cache_hit_rate',name:t('缓存命中率'),color:'#e56b8a',right:true,dashed:true,format:v=>Number(v).toFixed(1)+'%'}],t('当前筛选条件下暂无 Token 趋势')); lineChart('costTrend',costPoints,[{key:'cost_micro_usd',name:t('费用'),color:'#1eb787',fill:'rgba(30,183,135,.13)',format:usd}],t('当前筛选条件下暂无费用趋势'),true); bindTrendChips('tokenTrendTotal','tokenTrend'); bindTrendChips('costTrendTotal','costTrend'); renderModelShare(models); renderModelRank(models); requestAnimationFrame(()=>requestAnimationFrame(()=>Object.values(charts).forEach(instance=>instance.resize()))); }
  function renderRecentPagination(pagination) { const p=pagination||{}, page=Number(p.page||1), size=Number(p.page_size||recentPageSize), total=Number(p.total||0), pages=Math.max(Number(p.total_pages||0),1), start=total?(page-1)*size+1:0, end=Math.min(page*size,total); recentPage=page; recentPageSize=size; const summary=t('显示')+' '+count(start)+'–'+count(end)+'，'+t('共')+' '+count(total)+t('条'); $('recentState').textContent=summary; $('recentPagination').innerHTML='<div class="page-meta"><span class="page-summary">'+summary+'</span><label>'+esc(t('每页'))+' <select id="recentPageSize"><option value="25">25'+esc(t('条'))+'</option><option value="50">50'+esc(t('条'))+'</option><option value="100">100'+esc(t('条'))+'</option><option value="200">200'+esc(t('条'))+'</option></select></label></div><div class="page-nav"><button type="button" id="recentPrev" '+(page<=1?'disabled':'')+'>'+uiIcon('chevronL')+esc(t('上一页'))+'</button><span>'+esc(t('第'))+' '+count(page)+' / '+count(pages)+' '+esc(t('页'))+'</span><button type="button" id="recentNext" '+(page>=pages?'disabled':'')+'>'+esc(t('下一页'))+uiIcon('chevronR')+'</button></div>'; initCustomControls($('recentPagination')); $('recentPageSize').value=String(size); refreshCustomControl($('recentPageSize')); $('recentPageSize').addEventListener('change',event=>{recentPageSize=Number(event.target.value);recentPage=1;load(false,false,true)}); $('recentPrev').addEventListener('click',()=>{recentPage=Math.max(1,page-1);load(false,false,true)}); $('recentNext').addEventListener('click',()=>{recentPage=Math.min(pages,page+1);load(false,false,true)}); }
  function rememberedToken() { try { return localStorage.getItem(LOOKUP_REMEMBER_KEY)||''; } catch(_) { return ''; } }
  function persistLookupToken(value, remember) { try { if (remember && value) localStorage.setItem(LOOKUP_REMEMBER_KEY, value); else localStorage.removeItem(LOOKUP_REMEMBER_KEY); } catch(_) {} }
  function showLoginGate(locked) { delete document.documentElement.dataset.lookupAuthed; const gate=$('loginGate'); gate.classList.remove('hidden'); gate.classList.toggle('is-locked', !!locked); requestAnimationFrame(()=>$('key').focus()); }
  function hideLoginGate() { $('loginGate').classList.add('hidden'); $('loginGate').classList.remove('is-locked'); }
  function render(result) { data=result; hideLoginGate(); const landing=$('landingHero'); if(landing) landing.classList.add('hidden'); const key=result.key, overview=result.overview, summary=result.usage_summary||{}, models=result.by_model||[]; $('keyLabel').textContent=key.label||t('我的密钥'); $('fingerprint').textContent=t('指纹')+' '+(key.fingerprint||'—'); $('quotas').innerHTML=[quota('总额度',Number(key.settled_spend_micro_usd||0)+Number(key.held_amount_micro_usd||0),key.quota_micro_usd,'累计已结算与在途预占',true),quota('日额度',overview.daily_micro_usd,key.daily_quota_micro_usd,'UTC 自然日'),quota('周额度',overview.weekly_micro_usd,key.weekly_quota_micro_usd,'UTC 周一开始'),quota('月额度',overview.monthly_micro_usd,key.monthly_quota_micro_usd,'UTC 自然月'),concurrent(overview.active_reservations,key.max_concurrent_requests)].join(''); renderModelTokenLimits(result); $('stats').innerHTML=[['activity',t('请求数'),count(summary.request_count)+' '+t('请求')],['layers','Token',tokenValue(Number(summary.total_tokens||0)||(Number(summary.input_tokens||0)+Number(summary.output_tokens||0)+Number(summary.reasoning_tokens||0)))],['coin',t('费用'),usd(summary.cost_micro_usd)],['bolt',t('在途请求'),count(overview.active_reservations)+' '+t('请求')]].map(([icon,label,value]) => '<div class="stat"><div class="label"><span class="stat-icon" aria-hidden="true">'+uiIcon(icon)+'</span>'+esc(label)+'</div><div class="value">'+esc(value)+'</div></div>').join(''); const modelSelect=$('model'), current=modelSelect.value; modelSelect.innerHTML='<option value="">'+esc(t('全部模型'))+'</option>'+models.map(x=>'<option value="'+esc(x.model)+'">'+esc(x.model)+'</option>').join(''); if ([...modelSelect.options].some(x=>x.value===current)) modelSelect.value=current; $('recent').innerHTML=table(['时间','模型名称','来源','结果','首字延迟','生成时间','TPS','思考强度','输入 Token','输出 Token','思考 Token','缓存读取 Token','缓存创建 Token','总 Token 数','缓存命中','费用'],(result.recent_usage||[]).map(x=>'<tr><td>'+esc(new Date(x.created_at).toLocaleString())+'</td><td class="model" title="'+esc(x.model)+'">'+esc(x.model)+'</td><td>'+esc(x.source||'—')+'</td><td>'+esc(x.result||'—')+'</td><td>'+esc(duration(x.first_token_latency_ms))+'</td><td>'+esc(duration(x.generation_duration_ms))+'</td><td>'+esc(tps(x.tokens_per_second))+'</td><td>'+esc(x.thinking_intensity||'—')+'</td><td>'+tokenValue(x.input_tokens)+'</td><td>'+tokenValue(x.output_tokens)+'</td><td>'+tokenValue(x.reasoning_tokens)+'</td><td>'+tokenValue(x.cache_read_tokens)+'</td><td>'+tokenValue(x.cache_creation_tokens)+'</td><td>'+tokenValue(allTokens(x))+'</td><td>'+esc(Number(x.cache_read_tokens||0)>0?t('是'):t('否'))+'</td><td>'+usd(x.cost_micro_usd)+'</td></tr>')); $('dashboard').classList.remove('hidden'); renderCharts(result); }
  function customControlText(select) { return select.selectedOptions[0] ? select.selectedOptions[0].textContent.trim() : t('请选择'); }
  function padDatePart(value) { return String(value).padStart(2, '0'); }
  function formatDateTimeLocal(date) {
    return date.getFullYear() + '-' + padDatePart(date.getMonth() + 1) + '-' + padDatePart(date.getDate()) + 'T' + padDatePart(date.getHours()) + ':' + padDatePart(date.getMinutes());
  }
  function parseDateTimeLocal(value) {
    if (!value) return null;
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? null : date;
  }
  function dispatchControlChange(control) { control.dispatchEvent(new Event('input', { bubbles:true })); control.dispatchEvent(new Event('change', { bubbles:true })); }
  function refreshCustomControl(control) {
    const wrapper = control.closest('.custom-control');
    if (!wrapper) return;
    const trigger = wrapper.querySelector('.custom-control-trigger');
    if (!trigger) return;
    trigger.disabled = control.disabled;
    if (control.tagName === 'SELECT') {
      trigger.querySelector('.custom-control-value').textContent = customControlText(control);
      wrapper.querySelectorAll('.custom-option').forEach(option => option.classList.toggle('selected', option.dataset.value === control.value));
      return;
    }
    const date = parseDateTimeLocal(control.value);
    trigger.querySelector('.custom-control-value').textContent = date
      ? new Intl.DateTimeFormat(locale || 'zh-CN', { year:'numeric', month:'2-digit', day:'2-digit', hour:'2-digit', minute:'2-digit', hour12:false }).format(date)
      : t('选择日期和时间');
  }
  function closeCustomControls(except) {
    document.querySelectorAll('.custom-control.open').forEach(wrapper => {
      if (wrapper === except) return;
      wrapper.classList.remove('open');
      const panel = wrapper.querySelector('.custom-control-panel');
      if (panel) panel.hidden = true;
    });
  }
  function buildCustomSelect(select) {
    if (select.closest('.custom-control')) return;
    const wrapper = document.createElement('span');
    wrapper.className = 'custom-control custom-select';
    const trigger = document.createElement('button');
    trigger.type = 'button';
    trigger.className = 'custom-control-trigger';
    trigger.setAttribute('aria-haspopup', 'listbox');
    const value = document.createElement('span');
    value.className = 'custom-control-value';
    trigger.append(value);
    const panel = document.createElement('div');
    panel.className = 'custom-control-panel custom-select-panel';
    panel.setAttribute('role', 'listbox');
    panel.hidden = true;
    select.before(wrapper);
    wrapper.append(select, trigger, panel);
    select.classList.add('native-control');
    const renderOptions = () => {
      panel.innerHTML = '';
      [...select.options].forEach(option => {
        const item = document.createElement('button');
        item.type = 'button';
        item.className = 'custom-option';
        item.dataset.value = option.value;
        item.textContent = option.textContent;
        item.disabled = option.disabled;
        item.classList.toggle('selected', option.selected);
        item.addEventListener('click', event => {
          event.stopPropagation();
          select.value = option.value;
          wrapper.classList.remove('open');
          panel.hidden = true;
          dispatchControlChange(select);
          refreshCustomControl(select);
        });
        panel.append(item);
      });
      refreshCustomControl(select);
    };
    trigger.addEventListener('click', () => {
      if (select.disabled) return;
      const opening = !wrapper.classList.contains('open');
      closeCustomControls(wrapper);
      wrapper.classList.toggle('open', opening);
      panel.hidden = !opening;
    });
    select.addEventListener('change', () => refreshCustomControl(select));
    new MutationObserver(renderOptions).observe(select, { childList:true, subtree:true, characterData:true });
    renderOptions();
  }
  function buildCustomDateInput(input) {
    if (input.closest('.custom-control')) return;
    const wrapper = document.createElement('span');
    wrapper.className = 'custom-control custom-date-control';
    const trigger = document.createElement('button');
    trigger.type = 'button';
    trigger.className = 'custom-control-trigger custom-date-trigger';
    trigger.setAttribute('aria-haspopup', 'dialog');
    const value = document.createElement('span');
    value.className = 'custom-control-value';
    trigger.append(value);
    const panel = document.createElement('div');
    panel.className = 'custom-control-panel custom-date-panel';
    panel.setAttribute('role', 'dialog');
    panel.hidden = true;
    input.before(wrapper);
    wrapper.append(input, trigger, panel);
    input.classList.add('native-control');
    let viewDate = parseDateTimeLocal(input.value) || new Date();
    const setDateValue = date => {
      const previous = parseDateTimeLocal(input.value);
      date.setHours(previous ? previous.getHours() : 0, previous ? previous.getMinutes() : 0, 0, 0);
      input.value = formatDateTimeLocal(date);
      dispatchControlChange(input);
      refreshCustomControl(input);
    };
    const renderCalendar = () => {
      const selected = parseDateTimeLocal(input.value);
      const year = viewDate.getFullYear();
      const month = viewDate.getMonth();
      const first = new Date(year, month, 1);
      const start = new Date(year, month, 1 - first.getDay());
      const today = new Date();
      const sameDay = (a, b) => a && b && a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
      const days = Array.from({ length:42 }, (_, index) => new Date(start.getFullYear(), start.getMonth(), start.getDate() + index));
      panel.innerHTML = '';
      const header = document.createElement('div');
      header.className = 'custom-date-header';
      const title = document.createElement('button');
      title.type = 'button';
      title.className = 'custom-date-title';
      title.textContent = new Intl.DateTimeFormat(locale || 'zh-CN', { year:'numeric', month:'long' }).format(new Date(year, month, 1));
      const nav = document.createElement('div');
      nav.className = 'custom-date-nav';
      [['‹', -1, t('上个月')], ['›', 1, t('下个月')]].forEach(([text, offset, label]) => {
        const button = document.createElement('button');
        button.type = 'button';
        button.textContent = text;
        button.setAttribute('aria-label', label);
        button.addEventListener('click', event => {
          event.stopPropagation();
          viewDate = new Date(year, month + offset, 1);
          renderCalendar();
        });
        nav.append(button);
      });
      header.append(title, nav);
      const weekdays = document.createElement('div');
      weekdays.className = 'custom-date-weekdays';
      const weekdayNames = new Intl.DateTimeFormat(locale || 'zh-CN', { weekday:'short' });
      Array.from({ length:7 }, (_, day) => weekdayNames.format(new Date(2024, 0, 7 + day))).forEach(day => { const item = document.createElement('span'); item.textContent = day; weekdays.append(item); });
      const dayGrid = document.createElement('div');
      dayGrid.className = 'custom-date-days';
      days.forEach(date => {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'custom-date-day';
        if (date.getMonth() !== month) button.classList.add('outside');
        if (sameDay(date, today)) button.classList.add('today');
        if (sameDay(date, selected)) button.classList.add('selected');
        button.textContent = date.getDate();
        button.addEventListener('click', () => { setDateValue(date); viewDate = new Date(date); renderCalendar(); });
        dayGrid.append(button);
      });
      const setTimePart = (part, amount) => {
        const date = parseDateTimeLocal(input.value) || new Date(year, month, 1);
        if (part === 'hours') date.setHours((date.getHours() + amount + 24) % 24);
        else date.setMinutes((date.getMinutes() + amount + 60) % 60);
        input.value = formatDateTimeLocal(date);
        dispatchControlChange(input);
        refreshCustomControl(input);
        renderCalendar();
      };
      const timeField = (part, value, amount) => {
        const field = document.createElement('div');
        field.className = 'custom-time-field';
        const decrement = document.createElement('button');
        decrement.type = 'button';
        decrement.textContent = '−';
        decrement.setAttribute('aria-label', part === 'hours' ? t('减少小时') : t('减少分钟'));
        decrement.addEventListener('click', event => { event.stopPropagation(); setTimePart(part, -amount); });
        const text = document.createElement('span');
        text.textContent = padDatePart(value);
        const increment = document.createElement('button');
        increment.type = 'button';
        increment.textContent = '+';
        increment.setAttribute('aria-label', part === 'hours' ? t('增加小时') : t('增加分钟'));
        increment.addEventListener('click', event => { event.stopPropagation(); setTimePart(part, amount); });
        field.append(decrement, text, increment);
        return field;
      };
      const time = document.createElement('div');
      time.className = 'custom-date-time';
      const timeLabel = document.createElement('span');
      timeLabel.textContent = t('时间');
      const fields = document.createElement('div');
      fields.className = 'custom-time-fields';
      const timeValue = selected || new Date(year, month, 1);
      const separator = document.createElement('span');
      separator.className = 'custom-time-separator';
      separator.textContent = ':';
      fields.append(timeField('hours', timeValue.getHours(), 1), separator, timeField('minutes', timeValue.getMinutes(), 5));
      time.append(timeLabel, fields);
      const footer = document.createElement('div');
      footer.className = 'custom-date-footer';
      const clear = document.createElement('button');
      clear.type = 'button';
      clear.textContent = t('清除');
      clear.addEventListener('click', () => { input.value = ''; dispatchControlChange(input); refreshCustomControl(input); renderCalendar(); });
      const now = document.createElement('button');
      now.type = 'button';
      now.textContent = t('此刻');
      now.addEventListener('click', () => { const date = new Date(); viewDate = new Date(date); setDateValue(date); renderCalendar(); });
      footer.append(clear, now);
      panel.append(header, weekdays, dayGrid, time, footer);
    };
    trigger.addEventListener('click', () => {
      if (input.disabled) return;
      const opening = !wrapper.classList.contains('open');
      closeCustomControls(wrapper);
      wrapper.classList.toggle('open', opening);
      panel.hidden = !opening;
      if (opening) { viewDate = parseDateTimeLocal(input.value) || new Date(); renderCalendar(); }
    });
    input.addEventListener('change', () => refreshCustomControl(input));
    refreshCustomControl(input);
  }
  function initCustomControls(root) {
    (root || document).querySelectorAll('select:not(.native-control)').forEach(buildCustomSelect);
    (root || document).querySelectorAll('input[type="datetime-local"]:not(.native-control)').forEach(buildCustomDateInput);
  }
  function refreshCustomControls() { document.querySelectorAll('.native-control').forEach(refreshCustomControl); }
  function normalizeKey(value) { return String(value||'').replace(/[\s\u200B-\u200D\uFEFF]/g,''); }
  function validateKey(value) { if(!value) return '请输入密钥。'; if(!/^[\x20-\x7E]+$/.test(value)) return '密钥包含非英文字符或不可见字符，请重新复制完整的 tk- 开头密钥。'; if(!/^tk-[A-Za-z0-9-]+$/.test(value)) return '密钥格式不正确，请输入 tk- 开头的完整密钥。'; return ''; }
  async function load(isNew=false, resetPage=false, recentOnly=false) { const error=$('error'), button=isNew?$('submit'):$('refresh'); error.classList.add('hidden'); if(isNew){ token=normalizeKey($('key').value); $('key').value=token; const validation=validateKey(token); if(validation) { error.textContent=t(validation); error.classList.remove('hidden'); return; } } if(resetPage||isNew) recentPage=1; if(!token) return; button.disabled=true; button.textContent=t('加载中…'); try { const q=[rangeQuery(),$('model').value?'model='+encodeURIComponent($('model').value):'', 'grain=hour', 'page='+recentPage, 'page_size='+recentPageSize, recentOnly?'recent_only=1':''].filter(Boolean).join('&'); const response=await fetch(endpoint+(q?'?'+q:''),{headers:{Authorization:'Bearer '+token},cache:'no-store'}), result=await response.json(); if(!response.ok) throw new Error(result.error||t('查询失败，请检查密钥。')); if(isNew) { persistLookupToken(token, $('rememberKey').checked); if($('rememberKey').checked) document.documentElement.dataset.lookupAuthed='1'; else delete document.documentElement.dataset.lookupAuthed; } if(recentOnly) { $('recent').innerHTML=table(['时间','模型名称','来源','结果','首字延迟','生成时间','TPS','思考强度','输入 Token','输出 Token','思考 Token','缓存读取 Token','缓存创建 Token','总 Token 数','缓存命中','费用'],(result.recent_usage||[]).map(x=>'<tr><td>'+esc(new Date(x.created_at).toLocaleString())+'</td><td class="model" title="'+esc(x.model)+'">'+esc(x.model)+'</td><td>'+esc(x.source||'—')+'</td><td>'+esc(x.result||'—')+'</td><td>'+esc(duration(x.first_token_latency_ms))+'</td><td>'+esc(duration(x.generation_duration_ms))+'</td><td>'+esc(tps(x.tokens_per_second))+'</td><td>'+esc(x.thinking_intensity||'—')+'</td><td>'+tokenValue(x.input_tokens)+'</td><td>'+tokenValue(x.output_tokens)+'</td><td>'+tokenValue(x.reasoning_tokens)+'</td><td>'+tokenValue(x.cache_read_tokens)+'</td><td>'+tokenValue(x.cache_creation_tokens)+'</td><td>'+tokenValue(allTokens(x))+'</td><td>'+esc(Number(x.cache_read_tokens||0)>0?t('是'):t('否'))+'</td><td>'+usd(x.cost_micro_usd)+'</td></tr>')); } else { render(result); } renderRecentPagination(result.recent_pagination); } catch(err) { error.textContent=t(err.message||'查询失败，请稍后重试。'); error.classList.remove('hidden'); if(isNew) showLoginGate(!data); } finally { button.disabled=false; button.textContent=t(isNew?'查询':'刷新统计'); } }
  function syncPrefMenus() {
    document.querySelectorAll('#langPanel [data-locale]').forEach(button=>button.classList.toggle('active', button.dataset.locale===locale));
    document.querySelectorAll('#themePanel [data-theme-value]').forEach(button=>button.classList.toggle('active', button.dataset.themeValue===theme));
    $('langBtn').setAttribute('aria-expanded', String(!$('langPanel').hidden));
    $('themeBtn').setAttribute('aria-expanded', String(!$('themePanel').hidden));
  }
  function closePrefMenus() { $('langPanel').hidden=true; $('themePanel').hidden=true; syncPrefMenus(); }
  function applyLocale(next) {
    locale = LOCALES.includes(next) ? next : 'zh-CN';
    document.documentElement.lang = locale;
    document.title = t('密钥自助查询');
    translateTree(document.body);
    refreshCustomControls();
    syncPrefMenus();
    if (data) { render(data); renderRecentPagination(data.recent_pagination); }
  }
  function applyTheme(next) {
    theme = THEMES.includes(next) ? next : 'auto';
    const palette = theme==='auto' ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'white') : theme;
    document.documentElement.dataset.theme = theme;
    document.documentElement.dataset.palette = palette;
    syncPrefMenus();
    Object.values(charts).forEach(instance=>instance.resize());
  }
  function setLocale(next) { persistCPA(CPA_LOCALE_KEY,'language',next); applyLocale(next); }
  function setTheme(next) { persistCPA(CPA_THEME_KEY,'theme',next); applyTheme(next); }
  function initPreferences() {
    locale = localeFromCPA();
    theme = themeFromCPA();
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', ()=>{ if (theme==='auto') applyTheme('auto'); });
    window.addEventListener('storage', event=>{ if (event.key===CPA_LOCALE_KEY) applyLocale(localeFromCPA()); if (event.key===CPA_THEME_KEY) applyTheme(themeFromCPA()); });
    window.setInterval(()=>{ const nextLocale=localeFromCPA(), nextTheme=themeFromCPA(); if (nextLocale!==locale) applyLocale(nextLocale); if (nextTheme!==theme) applyTheme(nextTheme); }, 250);
    translationObserver = new MutationObserver(records=>records.forEach(record=>record.addedNodes.forEach(node=>translateTree(node))));
    translationObserver.observe(document.body, { childList:true, subtree:true });
    applyTheme(theme);
    applyLocale(locale);
    syncTrendGrainSwitch();
  }
  $('switchKey').addEventListener('click',()=>{ showLoginGate(false); $('key').select(); });
  $('loginGate').addEventListener('click',event=>{ if(event.target!==$('loginGate')||$('loginGate').classList.contains('is-locked')||!data) return; hideLoginGate(); $('dashboard').classList.remove('hidden'); });
  $('lookupForm').addEventListener('submit',e=>{e.preventDefault();load(true)}); $('refresh').addEventListener('click',()=>load(false,true)); $('range').addEventListener('change',()=>{syncCustomRange(); closeCustomControls(); tokenTrendGrain=costTrendGrain=defaultTrendGrain($('range').value); syncTrendGrainSwitch(); if($('range').value!=='custom')load(false,true)}); document.querySelectorAll('.trend-grain-switch [data-trend-grain]').forEach(button=>button.addEventListener('click',()=>setTrendGrain(button.closest('.trend-grain-switch').dataset.trendChart, button.dataset.trendGrain))); $('model').addEventListener('change',()=>load(false,true)); $('from').addEventListener('change',queueCustomRangeLoad); $('to').addEventListener('change',queueCustomRangeLoad); document.querySelectorAll('#tokenUnitSwitch [data-token-unit]').forEach(button=>button.addEventListener('click',()=>{tokenUnit=button.dataset.tokenUnit;localStorage.setItem(TOKEN_UNIT_KEY,tokenUnit);syncDisplayControls();refreshDisplayUnits()})); document.querySelectorAll('#currencySwitch [data-currency]').forEach(button=>button.addEventListener('click',()=>{currency=button.dataset.currency;localStorage.setItem(CURRENCY_KEY,currency);syncDisplayControls();refreshDisplayUnits()}));  document.querySelectorAll('#modelMetric [data-metric]').forEach(button=>button.addEventListener('click',()=>{modelMetric=button.dataset.metric; if(data)renderCharts(data)})); document.querySelectorAll('#modelRankMetric [data-rank-metric]').forEach(button=>button.addEventListener('click',()=>{modelRankMetric=button.dataset.rankMetric==='unit'||button.dataset.rankMetric==='cache'||button.dataset.rankMetric==='tps'?button.dataset.rankMetric:'value'; if(data)renderModelRank(data.by_model||[])}));
  $('langBtn').addEventListener('click', event=>{ event.stopPropagation(); const open=$('langPanel').hidden; closePrefMenus(); $('langPanel').hidden=!open; syncPrefMenus(); });
  $('themeBtn').addEventListener('click', event=>{ event.stopPropagation(); const open=$('themePanel').hidden; closePrefMenus(); $('themePanel').hidden=!open; syncPrefMenus(); });
  $('langPanel').addEventListener('click', event=>{ const button=event.target.closest('[data-locale]'); if (!button) return; setLocale(button.dataset.locale); closePrefMenus(); });
  $('themePanel').addEventListener('click', event=>{ const button=event.target.closest('[data-theme-value]'); if (!button) return; setTheme(button.dataset.themeValue); closePrefMenus(); });
  document.addEventListener('click', closePrefMenus);
  document.addEventListener('click', event => { if (!event.target.closest('.custom-control')) closeCustomControls(); });
  initCustomControls();
  syncDisplayControls(); initPreferences(); fetchUsdCnyRate(false); window.setInterval(()=>fetchUsdCnyRate(false),1800000); window.addEventListener('resize',()=>Object.values(charts).forEach(instance=>instance.resize()));
  const saved=rememberedToken();
  if (saved) { $('key').value=saved; $('rememberKey').checked=true; load(true); } else { showLoginGate(true); }
})();
