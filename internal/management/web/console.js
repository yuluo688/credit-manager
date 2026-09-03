
(function () {
  const TOKEN_KEY = 'credit_manager_mgmt_token';
  const TOKEN_ORIGIN_KEY = 'credit_manager_mgmt_token_origin';
  const BASE_KEY = 'credit_manager_api_base';
  const CUSTOM_BASE_ENABLED_KEY = 'credit_manager_custom_api_enabled';
  const TOKEN_UNIT_KEY = 'credit_manager_token_unit';
  const CURRENCY_KEY = 'credit_manager_currency';
  const USD_CNY_RATE_KEY = 'credit_manager_usd_cny_rate';
  const CPA_LOCALE_KEY = 'cli-proxy-language';
  const CPA_THEME_KEY = 'cli-proxy-theme';
  const DEFAULT_USD_CNY_RATE = 7.2;
  const LOCALES = ['zh-CN', 'zh-TW', 'en', 'ru'];
  const THEMES = ['auto', 'white', 'light', 'dark'];
  const COPY = {
    '语言': { 'zh-TW':'語言', en:'Language', ru:'Язык' }, '主题': { 'zh-TW':'主題', en:'Theme', ru:'Тема' },
    '跟随系统': { 'zh-TW':'跟隨系統', en:'System', ru:'Системная' }, '纯白': { 'zh-TW':'純白', en:'White', ru:'Белая' }, '羊毛纸': { 'zh-TW':'羊毛紙', en:'Parchment', ru:'Пергамент' }, '暗色': { 'zh-TW':'暗色', en:'Dark', ru:'Тёмная' },
    '额度仪表盘': { 'zh-TW':'額度儀表板', en:'Credit Dashboard', ru:'Панель лимитов' }, '连接设置': { 'zh-TW':'連線設定', en:'Connection settings', ru:'Настройки подключения' }, '刷新数据': { 'zh-TW':'重新整理資料', en:'Refresh data', ru:'Обновить данные' },
     '概览': { 'zh-TW':'概覽', en:'Overview', ru:'Обзор' }, '密钥管理': { 'zh-TW':'密鑰管理', en:'Keys', ru:'Ключи' }, '模型与价格': { 'zh-TW':'模型與價格', en:'Models & pricing', ru:'Модели и цены' }, '使用统计': { 'zh-TW':'使用統計', en:'Usage analytics', ru:'Статистика использования' },
    '额度概览': { 'zh-TW':'額度概覽', en:'Credit overview', ru:'Обзор лимитов' }, '按时间、密钥、账号、模型和来源聚合账本数据。': { 'zh-TW':'依時間、密鑰、帳號、模型與來源彙總帳本資料。', en:'Aggregate ledger data by time, key, account, model, and source.', ru:'Агрегируйте данные журнала по времени, ключу, аккаунту, модели и источнику.' }, '账本运行中': { 'zh-TW':'帳本運行中', en:'Ledger online', ru:'Журнал работает' },
    '概览筛选': { 'zh-TW':'概覽篩選', en:'Overview filters', ru:'Фильтры обзора' }, '筛选仅影响概览指标和图表；「今日」按本地自然日。': { 'zh-TW':'篩選僅影響概覽指標和圖表；「今天」按本地自然日。', en:'Filters affect overview metrics and charts only; Today uses the local calendar day.', ru:'Фильтры влияют только на метрики и диаграммы; «Сегодня» — локальный день.' }, '今日 · 全部数据': { 'zh-TW':'今天 · 全部資料', en:'Today · all data', ru:'Сегодня · все данные' }, '最近 30 天 · 全部数据': { 'zh-TW':'最近 30 天 · 全部資料', en:'Last 30 days · all data', ru:'Последние 30 дней · все данные' },
    '时间范围': { 'zh-TW':'時間範圍', en:'Time range', ru:'Период' }, '今日': { 'zh-TW':'今天', en:'Today', ru:'Сегодня' }, '最近 7 天': { 'zh-TW':'最近 7 天', en:'Last 7 days', ru:'Последние 7 дней' }, '最近 30 天': { 'zh-TW':'最近 30 天', en:'Last 30 days', ru:'Последние 30 дней' }, '最近 90 天': { 'zh-TW':'最近 90 天', en:'Last 90 days', ru:'Последние 90 дней' }, '全部时间': { 'zh-TW':'全部時間', en:'All time', ru:'За всё время' }, '自定义范围': { 'zh-TW':'自訂範圍', en:'Custom range', ru:'Свой период' }, '时间范围（UTC）': { 'zh-TW':'時間範圍（UTC）', en:'Time range (UTC)', ru:'Период (UTC)' }, '至': { 'zh-TW':'至', en:'to', ru:'до' },
    '密钥': { 'zh-TW':'密鑰', en:'Key', ru:'Ключ' }, '全部密钥': { 'zh-TW':'全部密鑰', en:'All keys', ru:'Все ключи' }, '账号': { 'zh-TW':'帳號', en:'Account', ru:'Аккаунт' }, '搜索账号名称或 ID': { 'zh-TW':'搜尋帳號名稱或 ID', en:'Search account name or ID', ru:'Поиск аккаунта или ID' }, '模型': { 'zh-TW':'模型', en:'Model', ru:'Модель' }, '全部已使用模型': { 'zh-TW':'全部已使用模型', en:'All used models', ru:'Все использованные модели' }, '来源': { 'zh-TW':'來源', en:'Source', ru:'Источник' }, '全部来源': { 'zh-TW':'全部來源', en:'All sources', ru:'Все источники' },
    '重置': { 'zh-TW':'重設', en:'Reset', ru:'Сбросить' }, '刷新概览': { 'zh-TW':'重新整理概覽', en:'Refresh overview', ru:'Обновить обзор' }, 'Token 趋势': { 'zh-TW':'Token 趨勢', en:'Token trend', ru:'Динамика токенов' }, '费用趋势': { 'zh-TW':'費用趨勢', en:'Cost trend', ru:'Динамика расходов' },     '模型调用占比': { 'zh-TW':'模型呼叫占比', en:'Model usage share', ru:'Доля вызовов моделей' }, '模型效率排行': { 'zh-TW':'模型效率排行', en:'Model efficiency rank', ru:'Рейтинг эффективности' }, '模型效率指标': { 'zh-TW':'模型效率指標', en:'Efficiency metric', ru:'Метрика эффективности' }, '性价比': { 'zh-TW':'性價比', en:'Value', ru:'Выгода' }, '单次': { 'zh-TW':'單次', en:'Per call', ru:'За запрос' }, '吞吐': { 'zh-TW':'吞吐', en:'Throughput', ru:'Скорость' },     '按每美元产出 Token 排序，越高越省': { 'zh-TW':'依每美元產出 Token 排序，越高越省', en:'Ranked by tokens per dollar; higher is cheaper', ru:'По токенам на доллар; больше — выгоднее' }, '按每人民币产出 Token 排序，越高越省': { 'zh-TW':'依每人民幣產出 Token 排序，越高越省', en:'Ranked by tokens per yuan; higher is cheaper', ru:'По токенам на юань; больше — выгоднее' }, '平均每次请求费用，越低越省': { 'zh-TW':'平均每次請求費用，越低越省', en:'Average cost per request; lower is cheaper', ru:'Средняя цена запроса; меньше — выгоднее' }, '缓存读取占输入比例，越高越省': { 'zh-TW':'快取讀取佔輸入比例，越高越省', en:'Cache-read share of input; higher is cheaper', ru:'Доля чтения кэша во входе; больше — выгоднее' }, '平均生成速度，越高越快': { 'zh-TW':'平均生成速度，越高越快', en:'Average generation speed; higher is faster', ru:'Средняя скорость генерации; больше — быстрее' }, '当前筛选条件下暂无模型效率数据': { 'zh-TW':'目前篩選條件下暫無模型效率資料', en:'No model efficiency data for the current filters', ru:'Нет данных об эффективности моделей' }, '免费': { 'zh-TW':'免費', en:'Free', ru:'Бесплатно' }, '次': { 'zh-TW':'次', en:'calls', ru:'вызов.' }, '调用次数': { 'zh-TW':'呼叫次數', en:'Requests', ru:'Запросы' }, '占比': { 'zh-TW':'佔比', en:'Share', ru:'Доля' }, '费用': { 'zh-TW':'費用', en:'Cost', ru:'Стоимость' }, '入': { 'zh-TW':'入', en:'In', ru:'Вх.' }, '出': { 'zh-TW':'出', en:'Out', ru:'Исх.' }, '缓存': { 'zh-TW':'快取', en:'Cache', ru:'Кэш' }, '命中': { 'zh-TW':'命中', en:'Hit', ru:'Попад.' }, '未连接': { 'zh-TW':'未連線', en:'Disconnected', ru:'Нет связи' },
    '趋势时间维度': { 'zh-TW':'趨勢時間維度', en:'Trend interval', ru:'Интервал тренда' }, 'Token 趋势时间维度': { 'zh-TW':'Token 趨勢時間維度', en:'Token trend interval', ru:'Интервал токенов' }, '费用趋势时间维度': { 'zh-TW':'費用趨勢時間維度', en:'Cost trend interval', ru:'Интервал расходов' },     '时': { 'zh-TW':'時', en:'Hour', ru:'Час' }, '日': { 'zh-TW':'日', en:'Day', ru:'День' }, '周': { 'zh-TW':'週', en:'Week', ru:'Неделя' }, '月': { 'zh-TW':'月', en:'Month', ru:'Месяц' },
    '按密钥设置额度、启停状态和可用模型策略。': { 'zh-TW':'依密鑰設定額度、啟停狀態和可用模型策略。', en:'Set limits, status, and model access for each key.', ru:'Настройте лимиты, статус и доступ к моделям для каждого ключа.' }, '额度隔离': { 'zh-TW':'額度隔離', en:'Isolated limits', ru:'Изолированные лимиты' }, '添加密钥': { 'zh-TW':'新增密鑰', en:'Add key', ru:'Добавить ключ' }, '已有密钥': { 'zh-TW':'已有密鑰', en:'Existing keys', ru:'Существующие ключи' }, '额度与状态一览': { 'zh-TW':'額度與狀態一覽', en:'Limits and status', ru:'Лимиты и статус' },
    '模型与价格': { 'zh-TW':'模型與價格', en:'Models & pricing', ru:'Модели и цены' }, '文本模型按百万 Token 计价；纯出图模型按张计费，不能套用 Token 价。': { 'zh-TW':'文字模型按百萬 Token 計價；純出圖模型按張計費，不能套用 Token 價。', en:'Text models bill per million tokens; image models bill per image.', ru:'Текстовые модели тарифицируются за миллион токенов, генерация изображений — за картинку.' }, '定价规则': { 'zh-TW':'定價規則', en:'Pricing rules', ru:'Правила цен' }, '当前代理模型': { 'zh-TW':'目前代理模型', en:'Current proxy models', ru:'Текущие модели прокси' }, '加载全部模型': { 'zh-TW':'載入全部模型', en:'Load all models', ru:'Загрузить все модели' }, '加载当前代理公开的模型后，可设置价格，或启用/禁用单个模型。禁用后无法调用，也不会出现在客户端模型列表中。': { 'zh-TW':'載入目前代理公開的模型後，可設定價格，或啟用/停用單一模型。停用後無法呼叫，也不會出現在客戶端模型列表中。', en:'After loading proxy models, set prices or enable/disable a model. Disabled models cannot be called and are omitted from the client model list.', ru:'После загрузки моделей задайте цены или включите/отключите модель. Отключённые модели нельзя вызвать, и они не попадают в клиентский список.' },     '启用': { 'zh-TW':'啟用', en:'Enable', ru:'Включить' }, '禁用': { 'zh-TW':'停用', en:'Disable', ru:'Отключить' }, '状态': { 'zh-TW':'狀態', en:'Status', ru:'Статус' }, '模型已启用': { 'zh-TW':'模型已啟用', en:'Model enabled', ru:'Модель включена' }, '模型已禁用': { 'zh-TW':'模型已停用', en:'Model disabled', ru:'Модель отключена' },
    '计费方式': { 'zh-TW':'計費方式', en:'Billing mode', ru:'Режим тарифа' }, '按 Token（USD / 1M）': { 'zh-TW':'按 Token（USD / 1M）', en:'Per token (USD / 1M)', ru:'За токен (USD / 1M)' }, '按张（出图）': { 'zh-TW':'按張（出圖）', en:'Per image', ru:'За изображение' }, '每张 USD': { 'zh-TW':'每張 USD', en:'USD / image', ru:'USD / изображение' }, '出图': { 'zh-TW':'出圖', en:'Image', ru:'Картинка' },     '按张计费': { 'zh-TW':'按張計費', en:'Billed per image', ru:'За картинку' },
    '从请求、Token 到费用的可筛选账本视图。': { 'zh-TW':'從請求、Token 到費用的可篩選帳本檢視。', en:'A filterable ledger view from requests and tokens to costs.', ru:'Фильтруемый журнал от запросов и токенов до расходов.' }, '实时汇总': { 'zh-TW':'即時彙總', en:'Live summary', ru:'Сводка в реальном времени' }, '统计筛选': { 'zh-TW':'統計篩選', en:'Usage filters', ru:'Фильтры статистики' }, '应用筛选': { 'zh-TW':'套用篩選', en:'Apply filters', ru:'Применить фильтры' }, '清除筛选': { 'zh-TW':'清除篩選', en:'Clear filters', ru:'Очистить фильтры' }, '按密钥汇总': { 'zh-TW':'依密鑰彙總', en:'By key', ru:'По ключам' }, '按模型汇总': { 'zh-TW':'依模型彙總', en:'By model', ru:'По моделям' }, '最近明细': { 'zh-TW':'最近明細', en:'Recent activity', ru:'Последние записи' }, '执行器': { 'zh-TW':'執行器', en:'Executor', ru:'Исполнитель' },
    '关闭提示': { 'zh-TW':'關閉提示', en:'Close notification', ru:'Закрыть уведомление' }, '取消': { 'zh-TW':'取消', en:'Cancel', ru:'Отмена' }, '保存规则': { 'zh-TW':'儲存規則', en:'Save rule', ru:'Сохранить правило' }, '删除': { 'zh-TW':'刪除', en:'Delete', ru:'Удалить' }, '编辑': { 'zh-TW':'編輯', en:'Edit', ru:'Изменить' }, '复制密钥': { 'zh-TW':'複製密鑰', en:'Copy key', ru:'Копировать ключ' }, '管理密钥': { 'zh-TW':'管理密鑰', en:'Manage key', ru:'Управлять ключом' },
    'CLIProxyAPI 根地址': { 'zh-TW':'CLIProxyAPI 根位址', en:'CLIProxyAPI base URL', ru:'Базовый URL CLIProxyAPI' }, '宿主管理密钥（Bearer）': { 'zh-TW':'宿主管理金鑰（Bearer）', en:'Host management token (Bearer)', ru:'Токен управления хостом (Bearer)' }, '清除本地信息': { 'zh-TW':'清除本機資訊', en:'Clear local data', ru:'Очистить локальные данные' }, '连接并加载': { 'zh-TW':'連線並載入', en:'Connect and load', ru:'Подключиться и загрузить' },
    '创建密钥': { 'zh-TW':'建立密鑰', en:'Create key', ru:'Создать ключ' }, '保存策略': { 'zh-TW':'儲存策略', en:'Save policy', ru:'Сохранить политику' }, '确认轮换': { 'zh-TW':'確認輪換', en:'Confirm rotation', ru:'Подтвердить ротацию' }, '新增价格规则': { 'zh-TW':'新增價格規則', en:'Add pricing rule', ru:'Добавить правило цены' }, '编辑价格规则': { 'zh-TW':'編輯價格規則', en:'Edit pricing rule', ru:'Изменить правило цены' },
    '生成方式': { 'zh-TW':'產生方式', en:'Key material', ru:'Способ создания' },
    '密钥已启用': { 'zh-TW':'密鑰已啟用', en:'Key enabled', ru:'Ключ включён' },
    '密钥已禁用': { 'zh-TW':'密鑰已停用', en:'Key disabled', ru:'Ключ отключён' },
    '总额度（USD）': { 'zh-TW':'總額度（USD）', en:'Total quota (USD)', ru:'Общий лимит (USD)' }, '日额度（USD）': { 'zh-TW':'日額度（USD）', en:'Daily quota (USD)', ru:'Дневной лимит (USD)' }, '周额度（USD）': { 'zh-TW':'週額度（USD）', en:'Weekly quota (USD)', ru:'Недельный лимит (USD)' }, '月额度（USD）': { 'zh-TW':'月額度（USD）', en:'Monthly quota (USD)', ru:'Месячный лимит (USD)' }, '最大并发请求数': { 'zh-TW':'最大併發請求數', en:'Max concurrent requests', ru:'Макс. параллельных запросов' },
    '按密钥设置额度、启停状态、可用模型和 Token 数量限制。': { 'zh-TW':'依密鑰設定額度、啟停狀態、可用模型和 Token 數量限制。', en:'Set spend limits, status, model access, and token caps for each key.', ru:'Настройте лимиты, статус, модели и потолки токенов для каждого ключа.' },
    '模型 Token 限制': { 'zh-TW':'模型 Token 限制', en:'Model token limits', ru:'Лимиты токенов модели' }, '按模型设置日/周/月数量': { 'zh-TW':'依模型設定日/週/月數量', en:'Daily, weekly, and monthly caps per model', ru:'Дневные, недельные и месячные лимиты по модели' },
    '搜索或输入模型 ID / glob': { 'zh-TW':'搜尋或輸入模型 ID / glob', en:'Search or type a model ID / glob', ru:'Поиск или ввод ID модели / glob' },
    '使用': { 'zh-TW':'使用', en:'Use', ru:'Использовать' },
    '没有可添加的模型': { 'zh-TW':'沒有可新增的模型', en:'No models to add', ru:'Нет моделей для добавления' },
    '添加模型': { 'zh-TW':'新增模型', en:'Add model', ru:'Добавить модель' },
    '日/周/月未填写时选择「可用」或「无限制」。未匹配模型单独选择可用或禁用。': { 'zh-TW':'日/週/月未填寫時選擇「可用」或「無限制」。未匹配模型單獨選擇可用或停用。', en:'Empty day/week/month fields can be Available or Unlimited. Unmatched models are Available or Disabled.', ru:'Пустые поля дня/недели/месяца: доступно или без лимита. Несовпавшие модели: доступны или отключены.' },
    '未匹配模型': { 'zh-TW':'未匹配模型', en:'Unmatched models', ru:'Несовпавшие модели' },
    '未列入下方的模型可以调用，且不限制 Token。': { 'zh-TW':'未列入下方的模型可以呼叫，且不限制 Token。', en:'Models not listed below can be called with no token cap.', ru:'Модели вне списка можно вызывать без лимита токенов.' },
    '仅下方列出的模型可以调用；未匹配的模型会被拒绝。': { 'zh-TW':'僅下方列出的模型可以呼叫；未匹配的模型會被拒絕。', en:'Only listed models can be called; unmatched models are rejected.', ru:'Можно вызывать только модели из списка; остальные отклоняются.' },
    '暂无模型 Token 限制': { 'zh-TW':'暫無模型 Token 限制', en:'No model token limits', ru:'Нет лимитов токенов' },
    '添加模型后，可设置日 / 周 / 月 Token 上限': { 'zh-TW':'新增模型後，可設定日 / 週 / 月 Token 上限', en:'Add a model to set daily, weekly, and monthly token caps', ru:'Добавьте модель, чтобы задать лимиты токенов' },
    '未填则选可用或无限制': { 'zh-TW':'未填則選可用或無限制', en:'Empty: Available or Unlimited', ru:'Пусто: доступно или без лимита' },
    '未匹配可用': { 'zh-TW':'未匹配可用', en:'Unmatched allowed', ru:'Несовпавшие доступны' },
    '未匹配禁用': { 'zh-TW':'未匹配停用', en:'Unmatched blocked', ru:'Несовпавшие запрещены' },
    '日 Token': { 'zh-TW':'日 Token', en:'Daily tokens', ru:'Токены за день' }, '周 Token': { 'zh-TW':'週 Token', en:'Weekly tokens', ru:'Токены за неделю' }, '月 Token': { 'zh-TW':'月 Token', en:'Monthly tokens', ru:'Токены за месяц' },
    '可用': { 'zh-TW':'可用', en:'Available', ru:'Доступно' }, '无限制': { 'zh-TW':'無限制', en:'Unlimited', ru:'Без лимита' },
    '移除': { 'zh-TW':'移除', en:'Remove', ru:'Убрать' },
    '请输入模型 ID 或 glob': { 'zh-TW':'請輸入模型 ID 或 glob', en:'Enter a model ID or glob', ru:'Введите ID модели или glob' },
    '该模型已在 Token 限制列表中': { 'zh-TW':'該模型已在 Token 限制列表中', en:'This model is already in the token limit list', ru:'Эта модель уже в списке лимитов' },
     '认证额度': { 'zh-TW':'認證額度', en:'Auth quotas', ru:'Квоты авторизации' }, '每个认证同时显示 5 小时窗口和当前额度周，可在卡片内切换该账号的其他额度周，并设置最大并发。': { 'zh-TW':'每個認證同時顯示 5 小時窗口與目前額度週，可在卡片內切換該帳號的其他額度週，並設定最大併發。', en:'Each auth shows its 5-hour window and current quota week. Switch weeks and set max concurrency inside the card.', ru:'У каждой авторизации видно 5-часовое окно и текущую неделю квоты; в карточке можно сменить неделю и задать параллельность.' },      '最大并发': { 'zh-TW':'最大併發', en:'Max concurrent', ru:'Макс. параллельность' }, '在途': { 'zh-TW':'在途', en:'In flight', ru:'В полёте' }, '当前并发量': { 'zh-TW':'目前併發量', en:'Current concurrency', ru:'Текущая параллельность' },     '条': { 'zh-TW':'條', en:'items', ru:'шт.' }, '并发': { 'zh-TW':'併發', en:'Concurrency', ru:'Параллельность' }, '批量并发': { 'zh-TW':'批量併發', en:'Batch concurrency', ru:'Пакетная параллельность' }, '应用到本页': { 'zh-TW':'套用到本頁', en:'Apply to page', ru:'К странице' }, '应用到筛选': { 'zh-TW':'套用到篩選', en:'Apply to filters', ru:'К фильтру' }, '没有可更新的认证': { 'zh-TW':'沒有可更新的認證', en:'No auths to update', ru:'Нет авторизаций для обновления' }, '认证并发已更新': { 'zh-TW':'認證併發已更新', en:'Auth concurrency updated', ru:'Параллельность авторизации обновлена' }, '最大并发请求数，0 或不填为不限制': { 'zh-TW':'最大併發請求數，0 或不填為不限制', en:'Max concurrent requests; 0 or empty = unlimited', ru:'Макс. параллельных запросов; 0 или пусто — без лимита' }, '额度周': { 'zh-TW':'額度週', en:'Quota week', ru:'Неделя квоты' }, '5 小时': { 'zh-TW':'5 小時', en:'5 hours', ru:'5 часов' }, '周额度': { 'zh-TW':'週額度', en:'Weekly', ru:'Неделя' }, '当前费用': { 'zh-TW':'目前費用', en:'Used cost', ru:'Текущие расходы' }, '预估剩余': { 'zh-TW':'預估剩餘', en:'Est. remaining', ru:'Ост. расходы' },     '预计可用': { 'zh-TW':'預計可用', en:'Est. available', ru:'Прогноз доступно' }, '平台': { 'zh-TW':'平台', en:'Platform', ru:'Платформа' }, '全部平台': { 'zh-TW':'全部平台', en:'All platforms', ru:'Все платформы' }, '名称': { 'zh-TW':'名稱', en:'Name', ru:'Имя' }, '搜索账号或名称': { 'zh-TW':'搜尋帳號或名稱', en:'Search account or name', ru:'Поиск аккаунта или имени' }, '重新加载': { 'zh-TW':'重新載入', en:'Reload', ru:'Обновить' }, '加载中': { 'zh-TW':'載入中', en:'Loading', ru:'Загрузка' }, '未同步': { 'zh-TW':'未同步', en:'Not synced', ru:'Не синхронизировано' }, '认证额度已刷新': { 'zh-TW':'認證額度已刷新', en:'Auth quota refreshed', ru:'Квота авторизации обновлена' }, '认证额度已从缓存刷新': { 'zh-TW':'認證額度已從快取刷新', en:'Auth quotas reloaded from cache', ru:'Квоты авторизации загружены из кэша' }, '订阅类型': { 'zh-TW':'訂閱類型', en:'Subscription', ru:'Подписка' }, '刷新本页': { 'zh-TW':'重新整理本頁', en:'Refresh page', ru:'Обновить страницу' }, '本页认证额度已刷新': { 'zh-TW':'本頁認證額度已刷新', en:'This page of auth quotas refreshed', ru:'Квоты на странице обновлены' }, '当前没有可用的认证额度数据': { 'zh-TW':'目前沒有可用的認證額度資料', en:'No auth quota data available', ru:'Нет данных о квотах авторизации' }, '没有符合筛选条件的认证额度': { 'zh-TW':'沒有符合篩選條件的認證額度', en:'No auth quotas match the filters', ru:'Нет квот, подходящих под фильтры' }, '每页': { 'zh-TW':'每頁', en:'Per page', ru:'На странице' }, '上一页': { 'zh-TW':'上一頁', en:'Previous', ru:'Назад' }, '下一页': { 'zh-TW':'下一頁', en:'Next', ru:'Вперёд' },
    'CPA 额度管理': { 'zh-TW':'CPA 額度管理', en:'CPA Credit Manager', ru:'Менеджер лимитов CPA' }, '选择日期和时间': { 'zh-TW':'選擇日期和時間', en:'Select date and time', ru:'Выберите дату и время' }, '上个月': { 'zh-TW':'上個月', en:'Previous month', ru:'Предыдущий месяц' }, '下个月': { 'zh-TW':'下個月', en:'Next month', ru:'Следующий месяц' }, '减少小时': { 'zh-TW':'減少小時', en:'Decrease hours', ru:'Уменьшить часы' }, '减少分钟': { 'zh-TW':'減少分鐘', en:'Decrease minutes', ru:'Уменьшить минуты' }, '增加小时': { 'zh-TW':'增加小時', en:'Increase hours', ru:'Увеличить часы' }, '增加分钟': { 'zh-TW':'增加分鐘', en:'Increase minutes', ru:'Увеличить минуты' }, '时间': { 'zh-TW':'時間', en:'Time', ru:'Время' }, '清除': { 'zh-TW':'清除', en:'Clear', ru:'Очистить' }, '此刻': { 'zh-TW':'此刻', en:'Now', ru:'Сейчас' },
    'Token 显示单位': { 'zh-TW':'Token 顯示單位', en:'Token display unit', ru:'Единица отображения токенов' }, '原始数量': { 'zh-TW':'原始數量', en:'Raw count', ru:'Исходное количество' },
    '千 (×1,000)': { 'zh-TW':'千 (×1,000)', en:'Thousand (×1,000)', ru:'Тысячи (×1,000)' }, 'k (×1,000)': { 'zh-TW':'k (×1,000)', en:'k (×1,000)', ru:'k (×1,000)' }, '万 (×10,000)': { 'zh-TW':'萬 (×10,000)', en:'10 thousand (×10,000)', ru:'Десятки тысяч (×10,000)' }, 'w (×10,000)': { 'zh-TW':'w (×10,000)', en:'w (×10,000)', ru:'w (×10,000)' }, '百万 (×1,000,000)': { 'zh-TW':'百萬 (×1,000,000)', en:'Million (×1,000,000)', ru:'Миллионы (×1,000,000)' }, 'm (×1,000,000)': { 'zh-TW':'m (×1,000,000)', en:'m (×1,000,000)', ru:'m (×1,000,000)' },
    '个': { 'zh-TW':'個', en:'个', ru:'个' }, '千': { 'zh-TW':'千', en:'千', ru:'千' }, 'k': { 'zh-TW':'k', en:'k', ru:'k' }, '万': { 'zh-TW':'萬', en:'万', ru:'万' }, 'w': { 'zh-TW':'w', en:'w', ru:'w' }, '百万': { 'zh-TW':'百萬', en:'百万', ru:'百万' }, 'm': { 'zh-TW':'m', en:'m', ru:'m' },
    '实时美元兑人民币汇率': { 'zh-TW':'即時美元兌人民幣匯率', en:'Live USD to CNY rate', ru:'Курс USD к CNY' },
    '汇率获取失败，已使用上次汇率': { 'zh-TW':'匯率取得失敗，已使用上次匯率', en:'Could not refresh the rate; using the last saved value', ru:'Не удалось обновить курс; используется прошлое значение' },
    '不限制': { 'zh-TW':'不限制', en:'Unlimited', ru:'Без ограничений' },
    '已用': { 'zh-TW':'已用', en:'Used', ru:'Использовано' }, '剩余': { 'zh-TW':'剩餘', en:'Remaining', ru:'Осталось' }, '限额': { 'zh-TW':'限額', en:'Cap', ru:'Лимит' },
    '同步': { 'zh-TW':'同步', en:'Synced', ru:'Синхр.' }, '最新': { 'zh-TW':'最新', en:'Fresh', ru:'Актуально' }, '缓存过期': { 'zh-TW':'快取過期', en:'Stale', ru:'Устарело' }, '不可用': { 'zh-TW':'不可用', en:'Unavailable', ru:'Недоступно' },
    '未命名认证': { 'zh-TW':'未命名認證', en:'Unnamed auth', ru:'Без имени' }, '未知提供商': { 'zh-TW':'未知供應商', en:'Unknown provider', ru:'Неизвестный провайдер' },
    '该额度周暂无窗口': { 'zh-TW':'該額度週暫無窗口', en:'No windows in this quota week', ru:'Нет окон на этой неделе' }, '暂无额度周': { 'zh-TW':'暫無額度週', en:'No quota weeks', ru:'Нет недель квоты' },
    '当前': { 'zh-TW':'目前', en:'Current', ru:'Текущая' }, '显示': { 'zh-TW':'顯示', en:'Showing', ru:'Показано' }, '共': { 'zh-TW':'共', en:'of', ru:'из' }, '第': { 'zh-TW':'第', en:'Page', ru:'Стр.' }, '页': { 'zh-TW':'頁', en:'', ru:'' },
    '重置时间': { 'zh-TW':'重設時間', en:'Resets', ru:'Сброс' }, '请选择': { 'zh-TW':'請選擇', en:'Select', ru:'Выберите' }, '全部可用': { 'zh-TW':'全部可用', en:'All available', ru:'Все доступны' },
    '标签': { 'zh-TW':'標籤', en:'Label', ru:'Метка' }, '可用模型': { 'zh-TW':'可用模型', en:'Allowed models', ru:'Доступные модели' }, '密钥限额': { 'zh-TW':'密鑰限額', en:'Key limits', ru:'Лимиты ключа' }, '已用 / 剩余': { 'zh-TW':'已用 / 剩餘', en:'Used / remaining', ru:'Использовано / остаток' }, '操作': { 'zh-TW':'操作', en:'Actions', ru:'Действия' },
    '全部模型': { 'zh-TW':'全部模型', en:'All models', ru:'Все модели' }, '暂无密钥': { 'zh-TW':'暫無密鑰', en:'No keys', ru:'Нет ключей' }, '已删除': { 'zh-TW':'已刪除', en:'Deleted', ru:'Удалено' }, '(无标签)': { 'zh-TW':'(無標籤)', en:'(no label)', ru:'(без метки)' },
    '还没有密钥。点击右上角“添加密钥”创建第一个额度凭证。': { 'zh-TW':'還沒有密鑰。點擊右上角「新增密鑰」建立第一個額度憑證。', en:'No keys yet. Use Add key in the top right to create the first credential.', ru:'Ключей пока нет. Нажмите «Добавить ключ», чтобы создать первую учётную запись.' },
    '连接信息仅保存于当前浏览器会话。未勾选自定义 API 地址时只连接当前站点。': { 'zh-TW':'連線資訊僅保存在目前瀏覽器工作階段。未勾選自訂 API 位址時只連線目前網站。', en:'Connection details stay in this browser session only. The page only talks to the current site unless Custom API address is checked.', ru:'Данные подключения хранятся только в этом сеансе. Без галочки страница подключается только к текущему сайту.' },
    '自定义 API 地址': { 'zh-TW':'自訂 API 位址', en:'Custom API address', ru:'Свой адрес API' },
    '基础信息': { 'zh-TW':'基礎資訊', en:'Basics', ru:'Основное' }, '标签与额度': { 'zh-TW':'標籤與額度', en:'Label and quotas', ru:'Метка и лимиты' },
    '可用模型（多选；不选=全部）': { 'zh-TW':'可用模型（多選；不選=全部）', en:'Allowed models (multi-select; none = all)', ru:'Доступные модели (несколько; пусто = все)' },
    '正在加载当前代理可用模型…': { 'zh-TW':'正在載入目前代理可用模型…', en:'Loading current proxy models…', ru:'Загрузка моделей прокси…' },
    '凭据设置': { 'zh-TW':'憑證設定', en:'Credential settings', ru:'Настройки ключа' }, '自定义密钥': { 'zh-TW':'自訂密鑰', en:'Custom key', ru:'Свой ключ' },
    '固定前缀无需输入；后缀格式为 <kid>-<secret>。': { 'zh-TW':'固定前綴無需輸入；後綴格式為 <kid>-<secret>。', en:'The prefix is fixed; suffix format is <kid>-<secret>.', ru:'Префикс фиксирован; суффикс: <kid>-<secret>.' },
    '生成': { 'zh-TW':'產生', en:'Generate', ru:'Создать' }, '查看密钥': { 'zh-TW':'查看密鑰', en:'View key', ru:'Показать ключ' }, '轮换密钥': { 'zh-TW':'輪換密鑰', en:'Rotate key', ru:'Сменить ключ' },
    '设置额度和模型策略后创建凭据。': { 'zh-TW':'設定額度和模型策略後建立憑證。', en:'Set quotas and model policy, then create the credential.', ru:'Задайте лимиты и политику моделей, затем создайте ключ.' },
    '密钥明文已加密保存，可在已有密钥中再次查看。': { 'zh-TW':'密鑰明文已加密儲存，可在已有密鑰中再次查看。', en:'The key plaintext is stored encrypted and can be viewed again under existing keys.', ru:'Открытый ключ хранится в шифре и его можно снова посмотреть в списке ключей.' },
    '规则标识': { 'zh-TW':'規則標識', en:'Rule identity', ru:'Идентификатор правила' }, 'ID 与参考价格': { 'zh-TW':'ID 與參考價格', en:'ID and reference price', ru:'ID и справочная цена' },
    '规则 ID': { 'zh-TW':'規則 ID', en:'Rule ID', ru:'ID правила' }, '可用模型价格': { 'zh-TW':'可用模型價格', en:'Available model prices', ru:'Цены доступных моделей' },
    '加载后选择模型': { 'zh-TW':'載入後選擇模型', en:'Choose a model after loading', ru:'Выберите модель после загрузки' },
    '匹配条件': { 'zh-TW':'匹配條件', en:'Match conditions', ru:'Условия совпадения' }, '类型 / pattern / 优先级': { 'zh-TW':'類型 / pattern / 優先級', en:'Type / pattern / priority', ru:'Тип / pattern / приоритет' },
    '匹配类型': { 'zh-TW':'匹配類型', en:'Match type', ru:'Тип совпадения' }, 'priority（大优先）': { 'zh-TW':'priority（大優先）', en:'priority (higher first)', ru:'priority (больше — выше)' },
    '计费价格': { 'zh-TW':'計費價格', en:'Billing price', ru:'Тариф' }, 'Token 或按张': { 'zh-TW':'Token 或按張', en:'Tokens or per image', ru:'Токены или за картинку' },
    '特殊档位': { 'zh-TW':'特殊檔位', en:'Special tiers', ru:'Особые тарифы' }, '上下文 / fast / priority': { 'zh-TW':'上下文 / fast / priority', en:'Context / fast / priority', ru:'Контекст / fast / priority' },
    '结算按实际输入 Token 和响应里的 service_tier；未命中则用上方默认价。': { 'zh-TW':'結算依實際輸入 Token 和回應裡的 service_tier；未命中則用上方預設價。', en:'Billing uses actual input tokens and the response service_tier; otherwise the default rates above.', ru:'Счёт идёт по фактическим input-токенам и service_tier ответа; иначе базовый тариф.' },
    '添加上下文档位': { 'zh-TW':'新增上下文檔位', en:'Add context tier', ru:'Добавить контекстный тариф' }, '上下文档位': { 'zh-TW':'上下文檔位', en:'Context tier', ru:'Контекстный тариф' }, '上下文': { 'zh-TW':'上下文', en:'Context', ru:'Контекст' },
    '阈值 Token': { 'zh-TW':'閾值 Token', en:'Threshold tokens', ru:'Порог токенов' }, '档位': { 'zh-TW':'檔位', en:'Tier', ru:'Тариф' },
    '未填则用上方默认价': { 'zh-TW':'未填則用上方預設價', en:'Blank uses default rates above', ru:'Пустое — базовый тариф' }, '缓存读': { 'zh-TW':'快取讀', en:'Cache read', ru:'Чтение кэша' }, '缓存写': { 'zh-TW':'快取寫', en:'Cache write', ru:'Запись кэша' },
    '缓存读取 USD/1M': { 'zh-TW':'快取讀取 USD/1M', en:'Cache read USD/1M', ru:'Чтение кэша USD/1M' }, '缓存创建 USD/1M': { 'zh-TW':'快取建立 USD/1M', en:'Cache create USD/1M', ru:'Создание кэша USD/1M' },
    '缓存会计': { 'zh-TW':'快取會計', en:'Cache accounting', ru:'Учёт кэша' },
    '按模型自动（OpenAI 含缓存 / Claude 不含）': { 'zh-TW':'依模型自動（OpenAI 含快取 / Claude 不含）', en:'Auto by model (OpenAI includes cache / Claude excludes)', ru:'Авто по модели (OpenAI с кэшем / Claude без)' },
    'input 含缓存（OpenAI）': { 'zh-TW':'input 含快取（OpenAI）', en:'input includes cache (OpenAI)', ru:'input включает кэш (OpenAI)' },
    'input 不含缓存（Claude）': { 'zh-TW':'input 不含快取（Claude）', en:'input excludes cache (Claude)', ru:'input без кэша (Claude)' },
    '价格来源：models.dev。同步会保存尚未配置的当前模型价格，不会覆盖已有规则。': { 'zh-TW':'價格來源：models.dev。同步會儲存尚未設定的目前模型價格，不會覆蓋已有規則。', en:'Prices from models.dev. Sync saves unconfigured current model prices without overwriting existing rules.', ru:'Цены из models.dev. Синхронизация сохраняет незаданные цены и не перезаписывает правила.' },
    '同步当前模型价格': { 'zh-TW':'同步目前模型價格', en:'Sync current model prices', ru:'Синхронизировать цены моделей' }, '同步价格': { 'zh-TW':'同步價格', en:'Sync prices', ru:'Синхронизировать' },
    '回填价格': { 'zh-TW':'回填價格', en:'Fill from catalog', ru:'Подставить цену' }, '选择当前模型': { 'zh-TW':'選擇目前模型', en:'Select a model', ru:'Выберите модель' },
    '匹配': { 'zh-TW':'匹配', en:'Match', ru:'Совпадение' }, '默认价': { 'zh-TW':'預設價', en:'Default rates', ru:'Базовый тариф' },
    '按 Token': { 'zh-TW':'按 Token', en:'Per token', ru:'За токен' }, '按张': { 'zh-TW':'按張', en:'Per image', ru:'За картинку' },
    '自动': { 'zh-TW':'自動', en:'Auto', ru:'Авто' }, '空白即用默认价': { 'zh-TW':'空白即用預設價', en:'Blank uses default', ru:'Пустое — база' }, '保存': { 'zh-TW':'儲存', en:'Save', ru:'Сохранить' },
    '文本模型按 USD / 1M tokens；纯出图模型按张计费，不要套用 Token 价。': { 'zh-TW':'文字模型按 USD / 1M tokens；純出圖模型按張計費，不要套用 Token 價。', en:'Text models use USD / 1M tokens; image models bill per image, not token prices.', ru:'Текстовые модели — USD / 1M токенов; картинки — за штуку, не по токенам.' },
    '未加载': { 'zh-TW':'未載入', en:'Not loaded', ru:'Не загружено' }, '未设置筛选': { 'zh-TW':'未設定篩選', en:'No filters', ru:'Нет фильтров' },
    '汇总和明细共用以下条件。「今日」按本地自然日，金额单位为 USD。': { 'zh-TW':'彙總和明細共用以下條件。「今天」按本地自然日，金額單位為 USD。', en:'Summary and details share these filters. Today uses the local calendar day; amounts are USD.', ru:'Сводка и детали используют одни фильтры. «Сегодня» — локальный день; суммы в USD.' },
    '最低费用（USD）': { 'zh-TW':'最低費用（USD）', en:'Min cost (USD)', ru:'Мин. расход (USD)' }, '最高费用（USD）': { 'zh-TW':'最高費用（USD）', en:'Max cost (USD)', ru:'Макс. расход (USD)' },
    '最低 Token 数': { 'zh-TW':'最低 Token 數', en:'Min tokens', ru:'Мин. токенов' }, '最高 Token 数': { 'zh-TW':'最高 Token 數', en:'Max tokens', ru:'Макс. токенов' },
    '点击区块或模型名称下钻': { 'zh-TW':'點擊區塊或模型名稱下鑽', en:'Click a slice or model name to drill down', ru:'Нажмите сектор или имя модели' },
    '输入 / 输出 / 缓存读取 / 缓存命中率': { 'zh-TW':'輸入 / 輸出 / 快取讀取 / 快取命中率', en:'Input / output / cache read / cache hit rate', ru:'Вход / выход / чтение кэша / попадания' },
    '按日汇总实际费用': { 'zh-TW':'按日彙總實際費用', en:'Actual cost by day', ru:'Фактические расходы по дням' },
    '删除密钥': { 'zh-TW':'刪除密鑰', en:'Delete key', ru:'Удалить ключ' }, '确认删除': { 'zh-TW':'確認刪除', en:'Confirm delete', ru:'Подтвердить удаление' },
    '删除后该密钥将立即失效，历史使用统计、预占和审计记录会保留。': { 'zh-TW':'刪除後該密鑰將立即失效，歷史使用統計、預占和稽核記錄會保留。', en:'The key is revoked immediately. Usage stats, holds, and audit records are kept.', ru:'Ключ сразу отключается. Статистика, резервы и аудит сохраняются.' },
    '重置已用': { 'zh-TW':'重設已用', en:'Reset used', ru:'Сбросить расход' }, '重置全部已用': { 'zh-TW':'重設全部已用', en:'Reset all used', ru:'Сбросить весь расход' },
    '重置已用额度': { 'zh-TW':'重設已用額度', en:'Reset used quota', ru:'Сброс израсходованной квоты' }, '确认重置': { 'zh-TW':'確認重設', en:'Confirm reset', ru:'Подтвердить сброс' },
    '不会删除用量记录，只把勾选周期的已用清零，限额不变。日/周/月到下一个自然周期后仍按 UTC 日历切换。': { 'zh-TW':'不會刪除用量記錄，只把勾選週期的已用清零，限額不變。日/週/月到下一個自然週期後仍按 UTC 日曆切換。', en:'Usage history is kept. Checked periods start from zero; caps stay the same. Day/week/month return to the UTC calendar at the next boundary.', ru:'История сохраняется. Отмеченные периоды обнуляются, лимиты те же. День/неделя/месяц снова идут по UTC с следующей границы.' },
    '总额度已用': { 'zh-TW':'總額度已用', en:'Total used', ru:'Общий расход' }, '日额度已用': { 'zh-TW':'日額度已用', en:'Daily used', ru:'Дневной расход' }, '周额度已用': { 'zh-TW':'週額度已用', en:'Weekly used', ru:'Недельный расход' }, '月额度已用': { 'zh-TW':'月額度已用', en:'Monthly used', ru:'Месячный расход' },
    '请至少选择一项': { 'zh-TW':'請至少選擇一項', en:'Select at least one period', ru:'Выберите хотя бы один период' },
    '即将重置：': { 'zh-TW':'即將重設：', en:'Will reset: ', ru:'Будет сброшено: ' },
    '全部启用密钥': { 'zh-TW':'全部啟用密鑰', en:'all active keys', ru:'все активные ключи' },
    '把密钥': { 'zh-TW':'把密鑰', en:'keys', ru:'ключа' },
    '已用额度已重置': { 'zh-TW':'已用額度已重設', en:'Used quota reset', ru:'Расход сброшен' },
    '关闭重置确认': { 'zh-TW':'關閉重設確認', en:'Close reset confirmation', ru:'Закрыть подтверждение сброса' },
    '0 或留空 = 不限制': { 'zh-TW':'0 或留空 = 不限制', en:'0 or empty = unlimited', ru:'0 или пусто = без лимита' },
    'gpt-4o 或 * 或 .*': { 'zh-TW':'gpt-4o 或 * 或 .*', en:'gpt-4o or * or .*', ru:'gpt-4o или * или .*' },
    '人民币（按汇率换算）': { 'zh-TW':'人民幣（按匯率換算）', en:'Chinese yuan (converted)', ru:'Юань (по курсу)' },
    '例如 0.01': { 'zh-TW':'例如 0.01', en:'e.g. 0.01', ru:'напр. 0.01' }, '例如 http://127.0.0.1:8317': { 'zh-TW':'例如 http://127.0.0.1:8317', en:'e.g. http://127.0.0.1:8317', ru:'напр. http://127.0.0.1:8317' },
    '关闭密钥弹窗': { 'zh-TW':'關閉密鑰彈窗', en:'Close key dialog', ru:'Закрыть диалог ключа' }, '关闭价格规则弹窗': { 'zh-TW':'關閉價格規則彈窗', en:'Close pricing dialog', ru:'Закрыть диалог цен' },
    '关闭删除确认': { 'zh-TW':'關閉刪除確認', en:'Close delete confirmation', ru:'Закрыть подтверждение удаления' }, '关闭连接设置': { 'zh-TW':'關閉連線設定', en:'Close connection settings', ru:'Закрыть настройки подключения' },
    '开始时间': { 'zh-TW':'開始時間', en:'Start time', ru:'Начало' }, '结束时间': { 'zh-TW':'結束時間', en:'End time', ru:'Конец' },
    '搜索密钥标签或 ID': { 'zh-TW':'搜尋密鑰標籤或 ID', en:'Search key label or ID', ru:'Поиск метки или ID ключа' },
    '模型占比指标': { 'zh-TW':'模型佔比指標', en:'Model share metric', ru:'Метрика доли моделей' },
    '美元': { 'zh-TW':'美元', en:'US dollar', ru:'Доллар США' }, '费用显示货币': { 'zh-TW':'費用顯示貨幣', en:'Cost currency', ru:'Валюта стоимости' },
    '输入密钥后缀': { 'zh-TW':'輸入密鑰後綴', en:'Enter key suffix', ru:'Введите суффикс ключа' },
    '尚未连接到 CPA': { 'zh-TW':'尚未連線到 CPA', en:'Not connected to CPA', ru:'Нет подключения к CPA' },
    '暂无可展示的数据': { 'zh-TW':'暫無可展示的資料', en:'Nothing to show yet', ru:'Пока нет данных' },
    '连接 CPA 管理接口后，即可查看 Token、费用和模型调用趋势。': { 'zh-TW':'連線 CPA 管理介面後，即可查看 Token、費用和模型呼叫趨勢。', en:'Connect the CPA management API to see token, cost, and model trends.', ru:'Подключите API управления CPA, чтобы видеть токены, расходы и модели.' },
    '图表组件加载失败，请检查网络连接后刷新页面': { 'zh-TW':'圖表元件載入失敗，請檢查網路連線後重新整理頁面', en:'Chart component failed to load. Check the network and refresh.', ru:'Не удалось загрузить график. Проверьте сеть и обновите страницу.' },
    '当前筛选条件下暂无 Token 趋势': { 'zh-TW':'目前篩選條件下暫無 Token 趨勢', en:'No token trend for the current filters', ru:'Нет динамики токенов для текущих фильтров' },
    '当前筛选条件下暂无费用趋势': { 'zh-TW':'目前篩選條件下暫無費用趨勢', en:'No cost trend for the current filters', ru:'Нет динамики расходов для текущих фильтров' },
    '当前筛选条件下暂无模型使用数据': { 'zh-TW':'目前篩選條件下暫無模型使用資料', en:'No model usage for the current filters', ru:'Нет данных по моделям для текущих фильтров' },
    '当前筛选条件下暂无按密钥汇总': { 'zh-TW':'目前篩選條件下暫無依密鑰彙總', en:'No key totals for the current filters', ru:'Нет сводки по ключам' },
    '当前筛选条件下暂无按模型汇总': { 'zh-TW':'目前篩選條件下暫無依模型彙總', en:'No model totals for the current filters', ru:'Нет сводки по моделям' },
    '当前筛选条件下暂无使用明细': { 'zh-TW':'目前篩選條件下暫無使用明細', en:'No usage details for the current filters', ru:'Нет детализации использования' },
    '左右滑动查看完整汇总': { 'zh-TW':'左右滑動查看完整彙總', en:'Swipe sideways for the full summary', ru:'Проведите в сторону, чтобы увидеть сводку' },
    '左右滑动查看完整明细': { 'zh-TW':'左右滑動查看完整明細', en:'Swipe sideways for the full details', ru:'Проведите в сторону, чтобы увидеть детали' },
    '输入': { 'zh-TW':'輸入', en:'Input', ru:'Вход' }, '输出': { 'zh-TW':'輸出', en:'Output', ru:'Выход' }, '缓存读取': { 'zh-TW':'快取讀取', en:'Cache read', ru:'Чтение кэша' }, '缓存命中率': { 'zh-TW':'快取命中率', en:'Cache hit rate', ru:'Попадания в кэш' },
    '请求数': { 'zh-TW':'請求數', en:'Requests', ru:'Запросы' }, '模型名称': { 'zh-TW':'模型名稱', en:'Model name', ru:'Модель' }, '首字延迟': { 'zh-TW':'首字延遲', en:'First-token latency', ru:'Задержка первого токена' },
    '生成时间': { 'zh-TW':'生成時間', en:'Generation time', ru:'Время генерации' }, '思考强度': { 'zh-TW':'思考強度', en:'Thinking intensity', ru:'Интенсивность мышления' },
    '思考': { 'zh-TW':'思考', en:'Reasoning', ru:'Рассуждение' }, '缓存创建': { 'zh-TW':'快取建立', en:'Cache create', ru:'Создание кэша' }, '总 Token 数': { 'zh-TW':'總 Token 數', en:'Total tokens', ru:'Всего токенов' }, '缓存命中': { 'zh-TW':'快取命中', en:'Cache hit', ru:'Попадание в кэш' },
    '是': { 'zh-TW':'是', en:'Yes', ru:'Да' }, '否': { 'zh-TW':'否', en:'No', ru:'Нет' }, '未设置': { 'zh-TW':'未設定', en:'Not set', ru:'Не задано' }, '待保存': { 'zh-TW':'待儲存', en:'Unsaved', ru:'Не сохранено' }, '无价格': { 'zh-TW':'無價格', en:'No price', ru:'Нет цены' },
    '设置价格': { 'zh-TW':'設定價格', en:'Set price', ru:'Задать цену' }, 'models.dev 价格': { 'zh-TW':'models.dev 價格', en:'models.dev price', ru:'Цена models.dev' }, '当前规则': { 'zh-TW':'目前規則', en:'Current rule', ru:'Текущее правило' },
    '指定密钥': { 'zh-TW':'指定密鑰', en:'Selected key', ru:'Выбранный ключ' }, '密钥数 / 可用': { 'zh-TW':'密鑰數 / 可用', en:'Keys / active', ru:'Ключи / активные' },
    '筛选请求': { 'zh-TW':'篩選請求', en:'Filtered requests', ru:'Запросы фильтра' }, '筛选 Token': { 'zh-TW':'篩選 Token', en:'Filtered tokens', ru:'Токены фильтра' }, '筛选费用': { 'zh-TW':'篩選費用', en:'Filtered cost', ru:'Расход фильтра' }, '模型数量': { 'zh-TW':'模型數量', en:'Models', ru:'Модели' },
    '认证并发已更新': { 'zh-TW':'認證併發已更新', en:'Auth concurrency updated', ru:'Параллельность авторизации обновлена' },
    '最大并发必须是大于等于 0 的整数': { 'zh-TW':'最大併發必須是大於等於 0 的整數', en:'Max concurrency must be an integer ≥ 0', ru:'Параллельность должна быть целым ≥ 0' },
    '没有可更新的认证': { 'zh-TW':'沒有可更新的認證', en:'No auths to update', ru:'Нет авторизаций для обновления' },
    '条明细': { 'zh-TW':'條明細', en:'records', ru:'записей' },
    '输入 Token': { 'zh-TW':'輸入 Token', en:'Input tokens', ru:'Входные токены' },
    '输出 Token': { 'zh-TW':'輸出 Token', en:'Output tokens', ru:'Выходные токены' },
    '个模型': { 'zh-TW':'個模型', en:'models', ru:'модели' },
    '个密钥': { 'zh-TW':'個密鑰', en:'keys', ru:'ключа' },
    '可直接勾选多个模型；不选择任何模型表示全部模型可用。': { 'zh-TW':'可直接勾選多個模型；不選擇任何模型表示全部模型可用。', en:'Select multiple models. Selecting none means all models are allowed.', ru:'Можно выбрать несколько моделей; пустой выбор означает все.' },
    '未发现可用模型；请确认宿主管理密钥和上游认证文件。': { 'zh-TW':'未發現可用模型；請確認宿主管理金鑰和上游認證檔。', en:'No models found. Check the host management token and upstream auth files.', ru:'Модели не найдены. Проверьте токен управления и файлы авторизации.' }, '未发现可用模型；请确认宿主管理密钥、上游认证文件或 AI 提供商。': { 'zh-TW':'未發現可用模型；請確認宿主管理金鑰、上游認證檔或 AI 提供商。', en:'No models found. Check the host management token, auth files, or AI providers.', ru:'Модели не найдены. Проверьте токен, файлы авторизации или AI-провайдеров.' },
    '未找到匹配的密钥': { 'zh-TW':'未找到匹配的密鑰', en:'No matching keys', ru:'Ключи не найдены' },
    '未找到匹配的账号': { 'zh-TW':'未找到匹配的帳號', en:'No matching accounts', ru:'Аккаунты не найдены' },
    '未知账号': { 'zh-TW':'未知帳號', en:'Unknown account', ru:'Неизвестный аккаунт' },
    '按张计费，不能套用 Token 价': { 'zh-TW':'按張計費，不能套用 Token 價', en:'Billed per image; token prices do not apply', ru:'За картинку; цены токенов не применяются' },
    '未找到唯一匹配价格': { 'zh-TW':'未找到唯一匹配價格', en:'No unique matching price', ru:'Нет однозначной цены' },
    '选择已匹配价格的当前模型': { 'zh-TW':'選擇已匹配價格的目前模型', en:'Choose a current model with a matched price', ru:'Выберите текущую модель с ценой' },
    '尚未加载模型目录。点击“加载全部模型”后将同步当前代理模型和 models.dev 价格。': { 'zh-TW':'尚未載入模型目錄。點擊「載入全部模型」後將同步目前代理模型和 models.dev 價格。', en:'Model catalog is not loaded. Click Load all models to sync proxy models and models.dev prices.', ru:'Каталог моделей не загружен. Нажмите «Загрузить все модели».' },
    '出图模型': { 'zh-TW':'出圖模型', en:'Image model', ru:'Модель изображений' },
    '历史密钥': { 'zh-TW':'歷史密鑰', en:'Historical key', ru:'Старый ключ' },
    '筛选请求数': { 'zh-TW':'篩選請求數', en:'Filtered request count', ru:'Число запросов фильтра' },
    '小时': { 'zh-TW':'小時', en:'hours', ru:'часов' }, '自然日': { 'zh-TW':'自然日', en:'days', ru:'дней' }, '自然月': { 'zh-TW':'自然月', en:'months', ru:'месяцев' },
    '请求次数': { 'zh-TW':'請求次數', en:'Requests', ru:'Запросы' },
    '指定密钥': { 'zh-TW':'指定密鑰', en:'Selected key', ru:'Выбранный ключ' },

  };
  const textSources = new WeakMap();
  const attributeSources = new WeakMap();
  let translationObserver;
  const TOKEN_UNITS = {
    raw: { div: 1, suffix: '', maxFrac: 0 },
    qian: { div: 1e3, suffix: '千', maxFrac: 2 },
    k: { div: 1e3, suffix: 'k', maxFrac: 2 },
    wan: { div: 1e4, suffix: '万', maxFrac: 2 },
    w: { div: 1e4, suffix: 'w', maxFrac: 2 },
    baiwan: { div: 1e6, suffix: '百万', maxFrac: 2 },
    m: { div: 1e6, suffix: 'm', maxFrac: 2 },
  };
  const state = {
    overview: null,
    keys: [],
    authQuotas: null,
    authQuotaWeeks: {},
    authQuotaRefreshing: {},
    authQuotaProvider: '',
    authQuotaName: '',
    authQuotaPage: 1,
    authQuotaPageSize: 12,
    authQuotaPageRefreshing: false,
    allKeys: [],
    usedAuths: [],
    modelPrices: {},
    modelCatalogError: '',
    availableModels: [],
    usagePage: 1,
    usagePageSize: 50,
    usageSummary: null,
    usageRecent: null,
    deleteKeyID: '',
    resetSpendID: '',
    resetSpendAll: false,
    tokenUnit: 'raw',
    currency: 'USD',
    usdCnyRate: DEFAULT_USD_CNY_RATE,
    currentTab: 'overview',
    tabLoadSeq: 0,
    modelShareMetric: 'requests',
    modelRankMetric: 'value',
    tokenTrendGrain: 'day',
    costTrendGrain: 'day',
    charts: {},
    overviewShareObserver: null,
    locale: 'zh-CN',
    theme: 'auto',
  };

  const $ = (id) => document.getElementById(id);
  function parseStoredPreference(value) {
    if (!value) return '';
    try {
      const parsed = JSON.parse(value);
      return parsed && typeof parsed === 'object' ? (parsed.state || parsed).language || (parsed.state || parsed).theme || '' : parsed;
    } catch (_) { return value; }
  }

  function localeFromCPA() {
    const raw = parseStoredPreference(localStorage.getItem(CPA_LOCALE_KEY)) || navigator.language || 'zh-CN';
    const normalized = String(raw).replace('_', '-').toLowerCase();
    if (normalized.startsWith('zh-tw') || normalized.startsWith('zh-hant')) return 'zh-TW';
    if (normalized.startsWith('ru')) return 'ru';
    if (normalized.startsWith('en')) return 'en';
    return 'zh-CN';
  }

  function themeFromCPA() {
    const theme = String(parseStoredPreference(localStorage.getItem(CPA_THEME_KEY)) || 'auto').toLowerCase();
    return THEMES.includes(theme) ? theme : 'auto';
  }

  function buildCopyReverse() {
    const reverse = Object.create(null);
    Object.keys(COPY).forEach(key => {
      const row = COPY[key];
      if (!row) return;
      ['zh-TW', 'en', 'ru'].forEach(loc => {
        const val = row[loc];
        if (!val || val === key) return;
        if (!(val in reverse)) reverse[val] = key;
        else if (reverse[val] !== key) reverse[val] = null;
      });
    });
    return reverse;
  }
  let COPY_REVERSE = null;
  function copyReverse() {
    if (!COPY_REVERSE) COPY_REVERSE = buildCopyReverse();
    return COPY_REVERSE;
  }

  function canonicalSource(text) {
    const trimmed = String(text == null ? '' : text).trim();
    if (!trimmed) return trimmed;
    if (COPY[trimmed]) return trimmed;
    const mapped = copyReverse()[trimmed];
    return mapped || trimmed;
  }

  function t(source) {
    const key = canonicalSource(source);
    const locale = state.locale || 'zh-CN';
    if (locale === 'zh-CN') return key;
    return (COPY[key] && COPY[key][locale]) || key;
  }

  function uiIcon(name) {
    const s = 'viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.85" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"';
    const icons = {
      gauge: '<svg '+s+'><path d="M2.6 11.8a5.8 5.8 0 1 1 10.8 0"/><path d="M8 9.4 10.7 6.7"/><circle cx="8" cy="10" r="1" fill="currentColor" stroke="none"/><path d="M2.6 13.8h10.8"/></svg>',
      key: '<svg '+s+'><circle cx="5.6" cy="10.4" r="3.1"/><path d="m7.9 8.1 5.2-5.2M10.7 5.3l1.9 1.9M12.5 3.5l1.7 1.7"/></svg>',
      tag: '<svg '+s+'><path d="M2.5 7.2V3.4a.9.9 0 0 1 .9-.9h3.8a1.6 1.6 0 0 1 1.1.5l5.2 5.2a1.6 1.6 0 0 1 0 2.3l-3.6 3.6a1.6 1.6 0 0 1-2.3 0L3 8.3a1.6 1.6 0 0 1-.5-1.1z"/><circle cx="5.7" cy="5.7" r=".9" fill="currentColor" stroke="none"/></svg>',
      chart: '<svg '+s+'><path d="M2.4 13.3h11.2"/><path d="M5 13.2V8.2M8 13.2V4.4M11 13.2V6.6" stroke-width="2.1"/></svg>',
      shield: '<svg '+s+'><path d="M8 1.9 13 3.8v3.4c0 3.2-2 5.6-5 6.9-3-1.3-5-3.7-5-6.9V3.8z"/><path d="m5.9 7.8 1.5 1.5 2.7-3"/></svg>',
      plug: '<svg '+s+'><path d="M5.5 2.5v3M10.5 2.5v3M3.9 5.5h8.2v2.3a4.1 4.1 0 0 1-8.2 0z"/><path d="M8 11.9v1.6"/></svg>',
      refresh: '<svg '+s+' stroke-width="1.7"><path d="M13.2 8A5.2 5.2 0 1 1 11.7 4.3M13.4 1.8v2.9h-2.9"/></svg>',
      plus: '<svg '+s+'><path d="M8 3v10M3 8h10"/></svg>',
      copy: '<svg '+s+'><rect x="5.4" y="5.4" width="8" height="8" rx="1.8"/><path d="M10.6 5.4V4.1a1.6 1.6 0 0 0-1.6-1.6H4.1a1.6 1.6 0 0 0-1.6 1.6v4.9a1.6 1.6 0 0 0 1.6 1.6h1.3"/></svg>',
      trash: '<svg '+s+'><path d="M2.8 4.2h10.4M6.4 4V3a1 1 0 0 1 1-1h1.2a1 1 0 0 1 1 1v1M4.2 4.4l.6 8.2a1.4 1.4 0 0 0 1.4 1.3h3.6a1.4 1.4 0 0 0 1.4-1.3l.6-8.2M6.6 7v4M9.4 7v4"/></svg>',
      filter: '<svg '+s+'><path d="M2.5 3.5h11L9.3 8.4v3.4l-2.6 1.4V8.4z"/></svg>',
      coin: '<svg '+s+'><circle cx="8" cy="8" r="5.5"/><path d="M8 5v6M6.2 6.3h3.6M6.2 9.7h3.6"/></svg>',
      wallet: '<svg '+s+' stroke-width="1.5"><path d="M2.5 5.5a1.8 1.8 0 0 1 1.8-1.8h7.4a1.8 1.8 0 0 1 1.8 1.8v6a1.8 1.8 0 0 1-1.8 1.8H4.3a1.8 1.8 0 0 1-1.8-1.8z"/><path d="M9.5 8.5h4M2.5 5.5h9.2"/></svg>',
      trend: '<svg '+s+'><path d="m2.5 11 3.5-3.5 2.4 2.4 4.6-5"/><path d="M9.8 4.9h3.2v3.2"/></svg>',
      clock: '<svg '+s+'><circle cx="8" cy="8" r="5.6"/><path d="M8 5.2V8l2 1.4"/></svg>',
      calendar: '<svg '+s+'><rect x="2.5" y="3.5" width="11" height="10" rx="2"/><path d="M5.5 2v3M10.5 2v3M2.5 7h11"/></svg>',
      bolt: '<svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path d="M9.2 1.5 3.6 9h3.2l-.9 5.5L11.9 7H8.6z"/></svg>',
      chevronL: '<svg '+s+'><path d="M10 3.5 5.5 8l4.5 4.5"/></svg>',
      chevronR: '<svg '+s+'><path d="m6 3.5 4.5 4.5L6 12.5"/></svg>',
      activity: '<svg '+s+'><path d="M1.8 8.2h2.7l1.6-4.4 2.8 8 1.7-3.6h3.6"/></svg>',
      layers: '<svg '+s+'><path d="m8 2 5.6 2.9L8 7.8 2.4 4.9z"/><path d="m2.4 8 5.6 2.9L13.6 8M2.4 11.1l5.6 2.9 5.6-2.9"/></svg>',
      cpu: '<svg '+s+'><rect x="4" y="4" width="8" height="8" rx="1.6"/><rect x="6.4" y="6.4" width="3.2" height="3.2" rx=".8"/><path d="M6.5 4V2.2M9.5 4V2.2M6.5 13.8V12M9.5 13.8V12M4 6.5H2.2M4 9.5H2.2M13.8 6.5H12M13.8 9.5H12"/></svg>',
      pie: '<svg '+s+'><circle cx="8" cy="8" r="5.6"/><path fill="currentColor" stroke="none" d="M8 8V2.4A5.6 5.6 0 0 1 13.4 10.4Z"/></svg>',
      trophy: '<svg '+s+'><path d="M5.2 2.5h5.6v3.5a2.8 2.8 0 0 1-5.6 0z"/><path d="M5.2 3.6H3.3a1.9 1.9 0 0 0 1.9 3M10.8 3.6h1.9a1.9 1.9 0 0 1-1.9 3M8 8.8v2.3M5.5 13.4h5"/></svg>',
      list: '<svg '+s+'><path d="M5.4 4h8M5.4 8h8M5.4 12h8"/><circle cx="2.7" cy="4" r=".9" fill="currentColor" stroke="none"/><circle cx="2.7" cy="8" r=".9" fill="currentColor" stroke="none"/><circle cx="2.7" cy="12" r=".9" fill="currentColor" stroke="none"/></svg>',
      x: '<svg '+s+'><path d="m4 4 8 8M12 4l-8 8"/></svg>',
      gear: '<svg '+s+'><circle cx="8" cy="8" r="2.2"/><path d="M8 1.8v1.9M8 12.3v1.9M12.9 4.3l-1.6 1M4.7 10.7l-1.6 1M12.9 11.7l-1.6-1M4.7 5.3l-1.6-1M13.4 8h-1.9M4.5 8H2.6"/></svg>',
      database: '<svg '+s+'><ellipse cx="8" cy="3.6" rx="5.4" ry="1.9"/><path d="M2.6 3.6v8.8c0 1 2.4 1.9 5.4 1.9s5.4-.9 5.4-1.9V3.6M2.6 8c0 1 2.4 1.9 5.4 1.9s5.4-.9 5.4-1.9"/></svg>',
      info: '<svg '+s+'><circle cx="8" cy="8" r="5.8"/><path d="M8 7.4v3.4"/><circle cx="8" cy="5.2" r=".9" fill="currentColor" stroke="none"/></svg>'
    };
    return icons[name] || '';
  }

  function translateElement(element) {
    if (!element || element.closest('[data-no-i18n]')) return;
    const attrs = ['title', 'placeholder', 'aria-label'];
    attrs.forEach(name => {
      if (!element.hasAttribute(name)) return;
      let values = attributeSources.get(element);
      if (!values) { values = {}; attributeSources.set(element, values); }
      if (!(name in values)) values[name] = element.getAttribute(name);
      values[name] = canonicalSource(values[name]);
      element.setAttribute(name, t(values[name]));
    });
    const textNodes = [...element.childNodes].filter(node => node.nodeType === Node.TEXT_NODE && node.nodeValue.trim());
    if (textNodes.length) {
      let values = textSources.get(element);
      if (!values) { values = new Map(); textSources.set(element, values); }
      textNodes.forEach(node => {
        if (!values.has(node)) values.set(node, node.nodeValue);
        const stored = values.get(node);
        const leading = stored.match(/^\s*/)[0];
        const trailing = stored.match(/\s*$/)[0];
        const key = canonicalSource(stored.trim());
        values.set(node, leading + key + trailing);
        node.nodeValue = leading + t(key) + trailing;
      });
    }
  }

  function translateTree(root) {
    if (!root) return;
    if (root.nodeType === Node.ELEMENT_NODE) translateElement(root);
    if (root.querySelectorAll) root.querySelectorAll('*').forEach(translateElement);
  }

  function applyLocale(locale) {
    state.locale = LOCALES.includes(locale) ? locale : 'zh-CN';
    document.documentElement.lang = state.locale;
    document.title = t('CPA 额度管理');
    translateTree(document.body);
    refreshCustomControls();
    refreshDisplayUnits();
    const keyModal = $('keyModal');
    if (keyModal && keyModal.getAttribute('aria-hidden') !== 'true' && $('keyModalTokenLimits')) {
      const rotating = $('keyModalMode') && $('keyModalMode').value === 'rotate';
      renderKeyTokenLimits(collectModelTokenLimits(), rotating);
      setUnmatchedModelsMode(unmatchedModelsMode(), rotating);
    }
  }

  function applyTheme(theme) {
    state.theme = THEMES.includes(theme) ? theme : 'auto';
    const palette = state.theme === 'auto'
      ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'white')
      : state.theme;
    document.documentElement.dataset.theme = state.theme;
    document.documentElement.dataset.palette = palette;
    Object.values(state.charts).forEach(chart => chart.resize());
  }

  function initPreferences() {
    state.locale = localeFromCPA();
    state.theme = themeFromCPA();
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (state.theme === 'auto') applyTheme('auto');
    });
    window.addEventListener('storage', event => {
      if (event.key === CPA_LOCALE_KEY) applyLocale(localeFromCPA());
      if (event.key === CPA_THEME_KEY) applyTheme(themeFromCPA());
    });
    // CPA and this resource page share an origin, but storage events do not fire
    // in the document that makes a change. Polling also covers same-tab updates.
    window.setInterval(() => {
      const locale = localeFromCPA();
      const theme = themeFromCPA();
      if (locale !== state.locale) applyLocale(locale);
      if (theme !== state.theme) applyTheme(theme);
    }, 250);
    translationObserver = new MutationObserver(records => {
      records.forEach(record => record.addedNodes.forEach(node => translateTree(node)));
    });
    translationObserver.observe(document.body, { childList:true, subtree:true });
    applyTheme(state.theme);
    applyLocale(state.locale);
  }
  let flashTimer = 0;
  const clearFlash = () => {
    const el = $('flash');
    if (!el) return;
    el.className = 'flash';
    if (flashTimer) {
      clearTimeout(flashTimer);
      flashTimer = 0;
    }
  };
  const flash = (msg, ok) => {
    const el = $('flash');
    const text = $('flashMessage');
    if (!el || !text) return;
    text.textContent = msg == null ? '' : t(String(msg));
    el.className = 'flash show ' + (ok ? 'ok' : 'err');
    if (flashTimer) clearTimeout(flashTimer);
    // Success toasts auto-hide; errors stay longer so they can be read.
    flashTimer = setTimeout(clearFlash, ok ? 2600 : 5200);
  };

  function normalizeTokenUnit(unit) {
    return TOKEN_UNITS[unit] ? unit : 'raw';
  }

  function getTokenUnit() {
    return normalizeTokenUnit(state.tokenUnit || localStorage.getItem(TOKEN_UNIT_KEY) || 'raw');
  }

  function formatTokens(value) {
    const n = Number(value);
    if (!Number.isFinite(n)) return '—';
    const unit = getTokenUnit();
    if (unit === 'raw') return Math.round(n).toLocaleString();
    const cfg = TOKEN_UNITS[unit];
    const scaled = n / cfg.div;
    const abs = Math.abs(scaled);
    // Preserve small non-zero values when a large display unit is selected.
    const maxFrac = abs > 0 && abs < 0.01 ? 6 : (abs >= 100 ? 1 : cfg.maxFrac);
    return scaled.toLocaleString(undefined, { maximumFractionDigits: maxFrac }) + cfg.suffix;
  }

  function tokenTitle(value) {
    const n = Number(value);
    if (!Number.isFinite(n)) return '';
    return Math.round(n).toLocaleString() + ' tokens';
  }

  function syncTokenUnitSwitch() {
    const unit = getTokenUnit();
    state.tokenUnit = unit;
    document.querySelectorAll('#tokenUnitSwitch [data-token-unit]').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.tokenUnit === unit);
    });
  }

  function normalizeCurrency(currency) {
    return currency === 'CNY' ? 'CNY' : 'USD';
  }

  function getCurrency() {
    return normalizeCurrency(state.currency || localStorage.getItem(CURRENCY_KEY) || 'USD');
  }

  function normalizeUsdCnyRate(value) {
    const n = Number(value);
    return Number.isFinite(n) && n > 0 ? n : DEFAULT_USD_CNY_RATE;
  }

  function getUsdCnyRate() {
    return normalizeUsdCnyRate(state.usdCnyRate);
  }

  function currencyCode() {
    return getCurrency();
  }

  function microToUSD(value) {
    if (value == null || value === '') return null;
    const n = Number(value);
    return Number.isFinite(n) ? n / 1e6 : null;
  }

  function formatUSDAmount(usd) {
    return usd.toLocaleString(undefined, { maximumFractionDigits: 6 });
  }

  function formatCNYAmount(cny) {
    const abs = Math.abs(cny);
    const maxFrac = abs >= 100 ? 2 : (abs >= 1 ? 3 : 4);
    return cny.toLocaleString(undefined, { maximumFractionDigits: maxFrac });
  }

  // Display helper: ledger stores micro-USD; CNY is display-only via local rate.
  function formatMoney(value) {
    const usd = microToUSD(value);
    if (usd == null) return '—';
    if (getCurrency() === 'CNY') {
      return '¥' + formatCNYAmount(usd * getUsdCnyRate());
    }
    return '$' + formatUSDAmount(usd);
  }

  function moneyTitle(value) {
    const usd = microToUSD(value);
    if (usd == null) return '';
    const usdText = '$' + formatUSDAmount(usd);
    if (getCurrency() === 'CNY') {
      return usdText + ' · ¥' + formatCNYAmount(usd * getUsdCnyRate()) + ' @ ' + formatUsdCnyRate(getUsdCnyRate());
    }
    return usdText;
  }

  function syncCurrencySwitch() {
    const currency = getCurrency();
    state.currency = currency;
    document.querySelectorAll('#currencySwitch [data-currency]').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.currency === currency);
    });
    const rateField = $('usdCnyRateField');
    const rateValue = $('usdCnyRate');
    if (rateField) rateField.classList.toggle('hidden', currency !== 'CNY');
    if (rateValue) rateValue.textContent = formatUsdCnyRate(getUsdCnyRate());
  }

  function formatUsdCnyRate(value) {
    return normalizeUsdCnyRate(value).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  }

  function fxURL(refresh) {
    const path = String(location.pathname || '').replace(/\/+$/, '');
    const base = path.replace(/\/console$/, '');
    return base + '/fx/usd-cny' + (refresh ? '?refresh=1' : '');
  }

  function modelsDevURL(refresh) {
    const path = String(location.pathname || '').replace(/\/+$/, '');
    const base = path.replace(/\/console$/, '');
    return base + '/models-dev' + (refresh ? '?refresh=1' : '');
  }

  async function fetchModelsDevCatalog(refresh) {
    const res = await fetch(modelsDevURL(refresh), {
      cache: 'no-store',
      headers: { 'Accept': 'application/json' },
      signal: (typeof AbortSignal !== 'undefined' && AbortSignal.timeout) ? AbortSignal.timeout(12000) : undefined,
    });
    const data = await res.json().catch(() => null);
    if (!res.ok) throw new Error((data && data.error) || 'models.dev HTTP ' + res.status);
    const catalog = data && data.catalog && typeof data.catalog === 'object' && !Array.isArray(data.catalog) ? data.catalog : {};
    const error = String((data && data.error) || '').trim();
    if (!Object.keys(catalog).length) throw new Error(error || 'models.dev 价格目录不可用');
    return { catalog, source: (data && data.source) || '', cached: Boolean(data && data.cached) };
  }

  async function fetchUsdCnyRate(refresh) {
    const field = $('usdCnyRateField');
    const valueEl = $('usdCnyRate');
    try {
      const res = await fetch(fxURL(refresh), { cache: 'no-store', headers: { 'Accept': 'application/json' } });
      const data = await res.json();
      const next = Number(data && data.usd_to_cny);
      if (!res.ok || !Number.isFinite(next) || next <= 0) throw new Error((data && data.error) || '汇率获取失败');
      setUsdCnyRate(next);
      if (field) {
        const asOf = data.fetched_at ? new Date(data.fetched_at).toLocaleString() : '';
        field.title = t('实时美元兑人民币汇率') + (asOf ? ' · ' + asOf : '');
      }
    } catch (_) {
      if (valueEl) valueEl.textContent = formatUsdCnyRate(getUsdCnyRate());
      if (field) field.title = t('汇率获取失败，已使用上次汇率');
    }
  }

  function refreshDisplayUnits() {
    if (state.overview) {
      renderOverview(state.overview);
      renderKeys(state.overview.keys || state.keys);
      renderPricing((state.overview.pricing) || []);
    } else if (state.keys && state.keys.length) {
      renderKeys(state.keys);
    }
    if (state.usageSummary || state.usageRecent) {
      renderUsage(state.usageSummary || {}, state.usageRecent || {});
    }
    if (state.authQuotas) renderAuthQuotas();
  }

  function setTokenUnit(unit) {
    const next = normalizeTokenUnit(unit);
    state.tokenUnit = next;
    localStorage.setItem(TOKEN_UNIT_KEY, next);
    syncTokenUnitSwitch();
    refreshDisplayUnits();
  }

  function setCurrency(currency) {
    const next = normalizeCurrency(currency);
    state.currency = next;
    localStorage.setItem(CURRENCY_KEY, next);
    syncCurrencySwitch();
    refreshDisplayUnits();
  }

  function setUsdCnyRate(value) {
    const next = normalizeUsdCnyRate(value);
    state.usdCnyRate = next;
    localStorage.setItem(USD_CNY_RATE_KEY, String(next));
    syncCurrencySwitch();
    refreshDisplayUnits();
  }

  const microFromUSD = (v) => {
    if (v === '' || v == null) return null;
    const n = Number(v);
    if (!Number.isFinite(n) || n < 0) throw new Error('金额无效');
    return Math.round(n * 1e6);
  };
  const esc = (s) => String(s == null ? '' : s)
    .replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');

  function pageOrigin() {
    return String(location.origin || '').replace(/\/$/, '');
  }

  function isSameOriginBase(value) {
    const raw = String(value || '').trim();
    if (!raw) return true;
    try {
      return new URL(raw, location.origin).origin === location.origin;
    } catch (_) {
      return false;
    }
  }

  function customAPIBaseEnabled() {
    const box = $('customApiBase');
    return !!(box && box.checked);
  }

  function persistCustomAPIBaseEnabled(enabled) {
    try { localStorage.setItem(CUSTOM_BASE_ENABLED_KEY, enabled ? '1' : '0'); } catch (_) {}
  }

  function persistCustomAPIBase(value) {
    const base = String(value || '').trim().replace(/\/$/, '');
    if (!base) {
      sessionStorage.removeItem(BASE_KEY);
      try { localStorage.removeItem(BASE_KEY); } catch (_) {}
      return;
    }
    sessionStorage.setItem(BASE_KEY, base);
    try { localStorage.setItem(BASE_KEY, base); } catch (_) {}
  }

  function savedCustomAPIBase() {
    const session = (sessionStorage.getItem(BASE_KEY) || '').trim().replace(/\/$/, '');
    if (session) return session;
    try { return (localStorage.getItem(BASE_KEY) || '').trim().replace(/\/$/, ''); }
    catch (_) { return ''; }
  }

  function restoreCustomAPIBasePreference() {
    const box = $('customApiBase');
    if (!box) return;
    try { box.checked = localStorage.getItem(CUSTOM_BASE_ENABLED_KEY) === '1'; }
    catch (_) { box.checked = false; }
  }

  function assertSameOriginRequest(url) {
    if (customAPIBaseEnabled()) return;
    if (!isSameOriginBase(url)) {
      throw new Error('已拒绝向非同源地址发送管理密钥');
    }
  }

  function savedSessionToken() {
    const savedOrigin = (sessionStorage.getItem(TOKEN_ORIGIN_KEY) || '').trim();
    if (savedOrigin && savedOrigin !== location.origin) return '';
    return (sessionStorage.getItem(TOKEN_KEY) || '').trim();
  }

  function token() {
    const typed = ($('mgmtToken') && $('mgmtToken').value || '').trim();
    if (typed) return typed;
    return savedSessionToken();
  }

  function detectDefaultBase() {
    if (!customAPIBaseEnabled()) return pageOrigin();
    return savedCustomAPIBase() || pageOrigin();
  }

  function apiBase() {
    if (!customAPIBaseEnabled()) return pageOrigin();
    const raw = (($('apiBase') && $('apiBase').value) || savedCustomAPIBase() || '').trim().replace(/\/$/, '');
    return raw || pageOrigin();
  }

  function syncAPIBaseField() {
    const input = $('apiBase');
    if (!input) return;
    const enabled = customAPIBaseEnabled();
    if (!enabled) {
      const typed = (input.value || '').trim().replace(/\/$/, '');
      if (!input.readOnly && typed && !isSameOriginBase(typed)) persistCustomAPIBase(typed);
      input.value = pageOrigin();
    } else {
      const saved = savedCustomAPIBase();
      if (saved) input.value = saved;
      else if (!(input.value || '').trim()) input.value = pageOrigin();
    }
    input.readOnly = !enabled;
  }

  function managementURL(path) {
    const base = apiBase();
    const p = '/v0/management/' + String(path || '').replace(/^\/+/, '');
    return base ? (base + p) : p;
  }

  function hostURL(path) {
    const base = apiBase();
    const p = '/' + String(path || '').replace(/^\/+/, '');
    return base ? (base + p) : p;
  }

  function errorMessage(data, text, status) {
    if (data) {
      if (typeof data.error === 'string' && data.error.trim()) return data.error.trim();
      if (data.error && typeof data.error === 'object') {
        if (typeof data.error.message === 'string' && data.error.message.trim()) return data.error.message.trim();
        if (typeof data.error.type === 'string' && data.error.type.trim()) return data.error.type.trim();
      }
      if (typeof data.message === 'string' && data.message.trim()) return data.message.trim();
    }
    if (text && String(text).trim()) return String(text).trim();
    return 'HTTP ' + status;
  }

  async function hostAPI(path) {
    const url = hostURL(path);
    assertSameOriginRequest(url);
    const res = await fetch(url, { headers: { 'Accept': 'application/json' } });
    const text = await res.text();
    let data;
    try { data = text ? JSON.parse(text) : null; } catch (_) { data = { raw: text }; }
    if (!res.ok) {
      throw new Error(errorMessage(data, text, res.status));
    }
    return data;
  }

  // Host management routes (/v0/management/*) use the remote-management secret,
  // not client API keys. Prefer this path for model discovery.
  async function hostManagementGET(path) {
    const t = token();
    if (!t) throw new Error('请先填写并保存宿主管理密钥');
    const url = managementURL(path);
    assertSameOriginRequest(url);
    let res;
    try {
      res = await fetch(url, {
        method: 'GET',
        headers: {
          'Authorization': 'Bearer ' + t,
          'Accept': 'application/json',
        },
      });
    } catch (e) {
      throw new Error('网络失败: ' + (e && e.message ? e.message : e) + '。请检查 API 根地址是否可访问。');
    }
    const text = await res.text();
    let data = null;
    try { data = text ? JSON.parse(text) : null; } catch (_) { data = { raw: text }; }
    if (!res.ok) {
      throw new Error(errorMessage(data, text, res.status));
    }
    return data;
  }

  async function fetchAvailableModels() {
    const fromManagement = await fetchModelsViaManagement();
    let fromV1 = [];
    let v1Error = null;
    try {
      fromV1 = modelIDs(await hostAPI('v1/models'));
    } catch (e) {
      v1Error = e;
    }
    const ids = [...new Set([...fromManagement, ...fromV1])].filter(Boolean).sort();
    if (ids.length) {
      return { data: ids.map(id => ({ id })), source: fromManagement.length ? 'management' : 'v1/models' };
    }
    if (v1Error) {
      const msg = v1Error && v1Error.message ? v1Error.message : String(v1Error);
      if (/missing api key|invalid api key|unauthorized|401/i.test(msg)) {
        throw new Error('无法通过管理接口发现模型，且 /v1/models 鉴权失败（' + msg + '）。请确认宿主已配置上游认证文件或 AI 提供商。');
      }
      throw v1Error;
    }
    return { data: [], source: 'management' };
  }

  async function fetchModelsViaManagement() {
    if (!token()) throw new Error('请先填写并保存宿主管理密钥');
    const ids = new Set();
    await Promise.all([
      collectAuthFileModels(ids),
      collectProviderKeyModels(ids),
      collectManagementCatalogModels(ids),
    ]);
    return [...ids].filter(Boolean).sort();
  }

  async function collectAuthFileModels(ids) {
    let filesPayload;
    try {
      filesPayload = await hostManagementGET('auth-files');
    } catch (_) {
      return;
    }
    const files = (filesPayload && filesPayload.files) || [];
    const names = [...new Set(files.map(file => {
      if (!file || typeof file !== 'object') return '';
      return String(file.name || file.id || file.Name || file.ID || '').trim();
    }).filter(Boolean))];
    if (!names.length) return;
    const results = await Promise.all(names.map(async name => {
      try {
        return await hostManagementGET('auth-files/models?name=' + encodeURIComponent(name));
      } catch (_) {
        return null;
      }
    }));
    results.forEach(payload => addCatalogModelIDs(ids, payload && payload.models));
  }

  async function collectProviderKeyModels(ids) {
    const routes = [
      'openai-compatibility',
      'gemini-api-key',
      'claude-api-key',
      'codex-api-key',
      'xai-api-key',
      'vertex-api-key',
      'interactions-api-key',
    ];
    const results = await Promise.all(routes.map(async route => {
      try {
        return { route, payload: await hostManagementGET(route) };
      } catch (_) {
        return null;
      }
    }));
    results.forEach(item => {
      if (!item) return;
      providerKeyEntries(item.payload, item.route).forEach(entry => addProviderEntryModels(ids, entry));
    });
  }

  async function collectManagementCatalogModels(ids) {
    try {
      const payload = await hostManagementGET('models?scope=available');
      addCatalogModelIDs(ids, payload && (payload.models || payload.data));
    } catch (_) {}
  }

  function providerKeyEntries(payload, route) {
    if (Array.isArray(payload)) return payload;
    if (!payload || typeof payload !== 'object') return [];
    for (const key of [route, 'items', 'list', 'data']) {
      if (Array.isArray(payload[key])) return payload[key];
    }
    return [];
  }

  function addProviderEntryModels(ids, entry) {
    if (!entry || typeof entry !== 'object' || entry.disabled) return;
    const prefix = String(entry.prefix || entry.Prefix || '').trim();
    const models = entry.models || entry.Models;
    if (!Array.isArray(models)) return;
    models.forEach(model => {
      if (typeof model === 'string') {
        addPrefixedModelID(ids, model, prefix);
        return;
      }
      if (!model || typeof model !== 'object') return;
      addPrefixedModelID(ids, model.alias || model.Alias, prefix);
      addPrefixedModelID(ids, model.name || model.Name || model.id || model.ID, prefix);
    });
  }

  function addCatalogModelIDs(ids, models) {
    if (!Array.isArray(models)) return;
    models.forEach(model => {
      const id = typeof model === 'string' ? model : (model && (model.id || model.ID || model.name || model.Name || model.alias || model.Alias));
      if (id) ids.add(String(id).trim());
    });
  }

  function addPrefixedModelID(ids, raw, prefix) {
    const id = String(raw || '').trim();
    if (!id) return;
    ids.add(id);
    const p = String(prefix || '').trim().replace(/\/+$/, '');
    if (p && id.indexOf(p + '/') !== 0) ids.add(p + '/' + id);
  }

  async function api(method, path, body) {
    const t = token();
    if (!t) throw new Error('请先填写并保存宿主管理密钥');
    const url = managementURL(path);
    assertSameOriginRequest(url);
    const opts = {
      method,
      headers: {
        'Authorization': 'Bearer ' + t,
        'Accept': 'application/json',
      },
    };
    if (body !== undefined) {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
    let res;
    try {
      res = await fetch(url, opts);
    } catch (e) {
      throw new Error('网络失败: ' + (e && e.message ? e.message : e) + '。请检查 API 根地址是否可访问，以及是否跨域被浏览器拦截。');
    }
    const text = await res.text();
    let data = null;
    try { data = text ? JSON.parse(text) : null; } catch (_) { data = { raw: text }; }
    if (!res.ok) {
      const err = (data && (data.error || data.message)) || text || res.statusText || ('HTTP ' + res.status);
      if (res.status === 404) {
        throw new Error('Not Found（404）：未找到管理接口 ' + url + '。请把「CLIProxyAPI 根地址」改成代理实际地址（如 http://127.0.0.1:8317），确认已部署最新 DLL 并重启宿主；密钥必须是宿主 management secret。');
      }
      if (res.status === 401 || res.status === 403) {
        throw new Error((typeof err === 'string' ? err : JSON.stringify(err)) + '（管理密钥错误，或未允许远程管理）');
      }
      throw new Error(typeof err === 'string' ? err : JSON.stringify(err));
    }
    return data;
  }

  function customControlText(select) {
    if (select.multiple) {
      const chosen = [...select.options].filter(option => option.selected).map(option => option.textContent.trim());
      return chosen.length ? chosen.join('、') : t('全部可用');
    }
    return select.selectedOptions[0] ? select.selectedOptions[0].textContent.trim() : t('请选择');
  }

  function padDatePart(value) {
    return String(value).padStart(2, '0');
  }

  function formatDateTimeLocal(date) {
    return date.getFullYear() + '-' + padDatePart(date.getMonth() + 1) + '-' + padDatePart(date.getDate()) + 'T' + padDatePart(date.getHours()) + ':' + padDatePart(date.getMinutes());
  }

  function formatDateTime(value) {
    if (!value) return '—';
    const date = value instanceof Date ? value : new Date(value);
    if (Number.isNaN(date.getTime())) return String(value);
    return date.getFullYear() + '-' + padDatePart(date.getMonth() + 1) + '-' + padDatePart(date.getDate()) + ' ' +
      padDatePart(date.getHours()) + ':' + padDatePart(date.getMinutes()) + ':' + padDatePart(date.getSeconds());
  }

  function parseDateTimeLocal(value) {
    if (!value) return null;
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? null : date;
  }

  function dispatchControlChange(control) {
    control.dispatchEvent(new Event('input', { bubbles: true }));
    control.dispatchEvent(new Event('change', { bubbles: true }));
  }

  function refreshCustomControl(control) {
    const wrapper = control.closest('.custom-control');
    if (!wrapper) return;
    const trigger = wrapper.querySelector('.custom-control-trigger');
    if (!trigger) return;
    trigger.disabled = control.disabled;
    if (control.tagName === 'SELECT') {
      trigger.querySelector('.custom-control-value').textContent = customControlText(control);
      wrapper.querySelectorAll('.custom-option').forEach(option => {
        option.classList.toggle('selected', option.dataset.value === control.value || (control.multiple && [...control.selectedOptions].some(selected => selected.value === option.dataset.value)));
      });
      return;
    }
    const date = parseDateTimeLocal(control.value);
    trigger.querySelector('.custom-control-value').textContent = date
      ? new Intl.DateTimeFormat(state.locale || 'zh-CN', { year:'numeric', month:'2-digit', day:'2-digit', hour:'2-digit', minute:'2-digit', hour12:false }).format(date)
      : t('选择日期和时间');
  }

  function closeCustomControls(except) {
    document.querySelectorAll('.custom-control.open').forEach(wrapper => {
      if (wrapper !== except) {
        wrapper.classList.remove('open');
        const panel = wrapper.querySelector('.custom-control-panel');
        if (panel) panel.hidden = true;
      }
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
        item.className = 'custom-option' + (select.multiple ? ' multi' : '');
        item.dataset.value = option.value;
        item.textContent = option.textContent;
        item.disabled = option.disabled;
        item.classList.toggle('selected', option.selected);
        item.addEventListener('click', event => {
          event.stopPropagation();
          if (select.multiple) {
            option.selected = !option.selected;
          } else {
            select.value = option.value;
            wrapper.classList.remove('open');
            panel.hidden = true;
          }
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
      title.textContent = new Intl.DateTimeFormat(state.locale || 'zh-CN', { year:'numeric', month:'long' }).format(new Date(year, month, 1));
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
      const weekdayNames = new Intl.DateTimeFormat(state.locale || 'zh-CN', { weekday:'short' });
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

  function refreshCustomControls() {
    document.querySelectorAll('.native-control').forEach(refreshCustomControl);
  }

  document.addEventListener('click', event => {
    if (!event.target.closest('.custom-control')) closeCustomControls();
  });
  document.addEventListener('click', event => {
    if (!event.target.closest('.key-search-control') && !event.target.closest('#keyModalTokenLimitOptions')) {
      closeKeySearch('overview');
      closeKeySearch('usage');
      closeAuthSearch('overview');
      closeAuthSearch('usage');
      closeTokenLimitModelSearch();
    }
  });
  document.addEventListener('click', event => {
    if (event.target.closest('.chart-connect-action')) openConnectionModal();
  });

  function setTab(name) {
    const tab = name || 'overview';
    document.querySelectorAll('.tab').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.tab === tab);
    });
    document.querySelectorAll('.tabpane').forEach(pane => {
      pane.classList.toggle('hidden', pane.id !== 'tab-' + tab);
    });
    state.currentTab = tab;
    if (tab === 'overview') requestAnimationFrame(resizeOverviewCharts);
    refreshActiveTab().catch(e => flash(e.message, false));
  }

  async function loadOverviewBundle() {
    const seq = state.tabLoadSeq;
    const data = await api('GET', 'credit-manager/overview' + overviewQuery());
    if (seq !== state.tabLoadSeq) return null;
    state.overview = data;
    renderKeys(data.keys || []);
    renderOverview(data);
    renderPricing(data.pricing || []);
    return data;
  }

  // Refresh only the data needed by the visible tab when switching.
  async function refreshActiveTab() {
    const tab = state.currentTab || 'overview';
    const seq = ++state.tabLoadSeq;
    if (tab === 'usage') {
      await loadOverviewBundle();
      if (seq !== state.tabLoadSeq) return;
      await loadUsage();
      if (seq !== state.tabLoadSeq) return;
      return;
    }
    if (tab === 'auth-quotas') {
      await loadAuthQuotas();
      return;
    }
    if (tab === 'pricing') {
      await loadOverviewBundle();
      if (seq !== state.tabLoadSeq) return;
      await loadModelCatalog();
      return;
    }
    // overview / keys
    await loadOverviewBundle();
  }

  function overviewRangeDates() {
    return presetRangeDates($('overviewRangeFilter').value);
  }

  function usageRangeDates() {
    return presetRangeDates($('usageRangeFilter').value);
  }

  function presetRangeDates(range) {
    if (range === 'all' || range === 'custom') return { from: '', to: '' };
    const to = new Date();
    if (range === 'today') {
      const from = new Date(to.getFullYear(), to.getMonth(), to.getDate());
      return { from: from.toISOString(), to: to.toISOString() };
    }
    const from = new Date(to.getTime());
    from.setUTCDate(from.getUTCDate() - Number(range));
    return { from: from.toISOString(), to: to.toISOString() };
  }

  function overviewDateInput(iso) {
    return iso ? new Date(iso).toISOString().slice(0, 16) : '';
  }

  function setOverviewRangeVisibility() {
    const custom = $('overviewRangeFilter').value === 'custom';
    $('overviewDateRangeField').classList.toggle('hidden', !custom);
    $('overviewFromFilter').disabled = !custom;
    $('overviewToFilter').disabled = !custom;
    if (!custom) {
      const range = overviewRangeDates();
      $('overviewFromFilter').value = overviewDateInput(range.from);
      $('overviewToFilter').value = overviewDateInput(range.to);
    }
    refreshCustomControls();
  }

  function setUsageRangeVisibility() {
    const custom = $('usageRangeFilter').value === 'custom';
    $('usageDateRangeField').classList.toggle('hidden', !custom);
    $('usageFromFilter').disabled = !custom;
    $('usageToFilter').disabled = !custom;
    if (!custom) {
      const range = usageRangeDates();
      $('usageFromFilter').value = overviewDateInput(range.from);
      $('usageToFilter').value = overviewDateInput(range.to);
    }
    refreshCustomControls();
  }

  function getOverviewFilterValues() {
    const value = id => ($(id).value || '').trim();
    const range = $('overviewRangeFilter').value === 'custom'
      ? { from: value('overviewFromFilter'), to: value('overviewToFilter') }
      : overviewRangeDates();
    const auth = resolveAuthFilter(value('overviewAuthFilter'));
    const filter = {
      plugin_key_id: resolveKeyFilter($('overviewKeyFilter')),
      auth_id: auth.auth_id,
      auth_provider: auth.auth_provider,
      auth_index: auth.auth_index,
      model: value('overviewModelFilter'),
      source: value('overviewSourceFilter'),
      from: range.from,
      to: range.to,
      limit: '5000',
    };
    if (filter.from) filter.from = new Date(filter.from).toISOString();
    if (filter.to) filter.to = new Date(filter.to).toISOString();
    if (filter.from && filter.to && filter.from > filter.to) throw new Error('开始时间不能晚于结束时间');
    return filter;
  }

  function overviewQuery() {
    const params = new URLSearchParams();
    Object.entries(getOverviewFilterValues()).forEach(([key, value]) => {
      if (value !== '') params.set(key, value);
    });
    return '?' + params.toString();
  }

  function usageTokens(item) {
    const reported = Number(item.total_tokens || 0);
    if (reported > 0) return reported;
    const total = Number(item.input_tokens || 0) + Number(item.output_tokens || 0) + Number(item.reasoning_tokens || 0);
    return total > 0 ? total : Number(item.cached_tokens || 0);
  }

  function overviewModelUsage(items) {
    const usage = new Map();
    items.forEach(item => {
      const name = item.model || '未标记模型';
      const current = usage.get(name) || {
        model: name, tokens: 0, requests: 0, cost: 0,
        cacheRead: 0, cacheRelated: 0, tpsSum: 0, tpsCount: 0,
      };
      current.tokens += usageTokens(item);
      current.requests += 1;
      current.cost += Number(item.cost_micro_usd || 0);
      current.cacheRead += Number(item.cache_read_tokens || 0);
      current.cacheRelated += usageCacheRelated(item);
      const tps = Number(item.tokens_per_second);
      if (Number.isFinite(tps) && tps > 0) {
        current.tpsSum += tps;
        current.tpsCount += 1;
      }
      usage.set(name, current);
    });
    return [...usage.values()].sort((a, b) => b.tokens - a.tokens || b.requests - a.requests);
  }

  const MODEL_SHARE_COLORS = ['#7c6cf0', '#3ecf8e', '#f0a04b', '#e88b9a', '#4dabf7', '#20c997', '#ff922b', '#cc5de8', '#51cf66', '#339af0'];

  function getModelShareMetric() {
    const metric = state.modelShareMetric || 'requests';
    return metric === 'tokens' || metric === 'cost' ? metric : 'requests';
  }

  function modelShareMetricValue(item, metric) {
    if (metric === 'tokens') return Number(item.tokens || 0);
    if (metric === 'cost') return Number(item.cost || 0);
    return Number(item.requests || 0);
  }

  function formatModelShareCenter(value, metric) {
    if (metric === 'tokens') return formatTokens(value);
    if (metric === 'cost') return formatMoney(value);
    return Number(value || 0).toLocaleString();
  }

  function modelShareCenterLabel(metric) {
    if (metric === 'tokens') return 'Token';
    if (metric === 'cost') return '费用';
    return '请求次数';
  }

  function syncModelShareMetricSwitch() {
    const metric = getModelShareMetric();
    document.querySelectorAll('#modelShareMetricSwitch [data-model-metric]').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.modelMetric === metric);
    });
  }

  function setModelShareMetric(metric) {
    state.modelShareMetric = metric === 'tokens' || metric === 'cost' ? metric : 'requests';
    syncModelShareMetricSwitch();
    if (state.overview) renderOverviewModels(state.overview.recent_usage || []);
  }

  function getModelRankMetric() {
    const metric = state.modelRankMetric || 'value';
    return metric === 'unit' || metric === 'cache' || metric === 'tps' ? metric : 'value';
  }

  function displaySpend(micro) {
    const usd = Number(micro || 0) / 1e6;
    return getCurrency() === 'CNY' ? usd * getUsdCnyRate() : usd;
  }

  function modelRankSpec(metric) {
    const key = metric || getModelRankMetric();
    if (key === 'unit') {
      return {
        key,
        label: '单次',
        hint: '平均每次请求费用，越低越省',
        higher: false,
        ready: item => item.requests > 0,
        score: item => item.requests > 0 ? item.cost / item.requests : NaN,
        format: value => formatMoney(value),
      };
    }
    if (key === 'cache') {
      return {
        key,
        label: '缓存',
        hint: '缓存读取占输入比例，越高越省',
        higher: true,
        ready: item => item.cacheRelated > 0,
        score: item => item.cacheRelated > 0 ? item.cacheRead / item.cacheRelated * 100 : NaN,
        format: value => Number(value).toFixed(1) + '%',
      };
    }
    if (key === 'tps') {
      return {
        key,
        label: '吞吐',
        hint: '平均生成速度，越高越快',
        higher: true,
        ready: item => item.tpsCount > 0,
        score: item => item.tpsCount > 0 ? item.tpsSum / item.tpsCount : NaN,
        format: value => Number(value).toFixed(value >= 10 ? 1 : 2) + ' tok/s',
      };
    }
    return {
      key: 'value',
      label: '性价比',
      hint: getCurrency() === 'CNY' ? '按每人民币产出 Token 排序，越高越省' : '按每美元产出 Token 排序，越高越省',
      higher: true,
      ready: item => item.tokens > 0,
      score: item => {
        const spend = displaySpend(item.cost);
        if (spend <= 0) return item.tokens > 0 ? Infinity : 0;
        return item.tokens / spend;
      },
      format: value => {
        if (!Number.isFinite(value)) return t('免费');
        return formatTokens(value) + (getCurrency() === 'CNY' ? '/¥' : '/$');
      },
    };
  }

  function syncModelRankMetricSwitch() {
    const metric = getModelRankMetric();
    document.querySelectorAll('#modelRankMetricSwitch [data-rank-metric]').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.rankMetric === metric);
    });
  }

  function setModelRankMetric(metric) {
    state.modelRankMetric = metric === 'unit' || metric === 'cache' || metric === 'tps' ? metric : 'value';
    syncModelRankMetricSwitch();
    if (state.overview) renderOverviewRank(state.overview.recent_usage || []);
  }

  async function drillOverviewModel(model) {
    const name = String(model || '').trim();
    if (!name) return;
    const select = $('overviewModelFilter');
    if (!select) return;
    if (![...select.options].some(opt => opt.value === name)) {
      const option = document.createElement('option');
      option.value = name;
      option.textContent = name;
      select.appendChild(option);
    }
    select.value = name;
    refreshCustomControl(select);
    try {
      await reload();
      flash('已下钻到模型：' + name, true);
    } catch (e) { flash(e.message, false); }
  }

  function usageCost(item) {
    return Number(item.cost_micro_usd || 0);
  }

  function usageCacheRelated(item) {
    const input = Number(item.input_tokens || 0);
    if (input > 0) return input;
    return Number(item.cache_read_tokens || item.cached_tokens || 0) + Number(item.cache_creation_tokens || 0);
  }

  function disposeOverviewChart(id) {
    const chart = state.charts[id];
    if (chart) chart.dispose();
    delete state.charts[id];
  }

  function chartEmptyState(message, disconnected, icon) {
    const title = disconnected ? '尚未连接到 CPA' : '暂无可展示的数据';
    const copy = disconnected
      ? '连接 CPA 管理接口后，即可查看 Token、费用和模型调用趋势。'
      : message;
    const action = disconnected ? '<button type="button" class="btn ghost chart-connect-action">连接设置</button>' : '';
    const glyph = disconnected ? 'plug' : (icon || 'chart');
    return '<div class="empty-state chart-empty-state">' +
      '<div><div class="chart-empty-icon" aria-hidden="true">'+uiIcon(glyph)+'</div>' +
      '<p class="chart-empty-title">'+esc(title)+'</p>' +
      '<p class="chart-empty-copy">'+esc(copy)+'</p>'+action+'</div></div>';
  }

  function renderDisconnectedOverview() {
    [['overviewTrend', 'chart'], ['overviewCostTrend', 'coin'], ['overviewModelShare', 'pie']].forEach(([id, icon]) => {
      const target = $(id);
      disposeOverviewChart(id);
      target.className = '';
      target.innerHTML = chartEmptyState('', true, icon);
    });
    renderOverviewModelShareLegend([]);
    const rank = $('overviewModelRank');
    if (rank) {
      rank.className = 'model-rank-list';
      rank.innerHTML = chartEmptyState('', true, 'trophy');
    }
    const stats = $('overviewStats');
    if (stats) stats.innerHTML = '';
    const filterState = $('overviewFilterState');
    if (filterState) {
      const range = $('overviewRangeFilter');
      filterState.textContent = (range && range.selectedOptions[0] && range.selectedOptions[0].textContent.trim()) || t('今日 · 全部数据');
    }
    const modelCount = $('overviewModelCount');
    if (modelCount) {
      modelCount.textContent = '';
      modelCount.hidden = true;
    }
    ['overviewTrendTotal', 'overviewCostTrendTotal', 'overviewModelShareTotal', 'overviewRankCount'].forEach(id => {
      const target = $(id);
      if (!target) return;
      target.textContent = t('未连接');
      if (id === 'overviewRankCount') target.hidden = true;
    });
    const rankHint = $('overviewRankHint');
    if (rankHint) rankHint.textContent = t(modelRankSpec('value').hint);
  }

  function renderDisconnectedTabStates() {
    const content = chartEmptyState('', true).replace('chart-empty-state', 'chart-empty-state tab-empty-state');
    ['keysTable', 'pricingTable', 'usageByKey', 'usageByModel', 'usageRecent', 'authQuotaList'].forEach(id => {
      const target = $(id);
      if (target) target.innerHTML = content;
    });
    const stats = $('usageStats');
    if (stats) stats.innerHTML = content;
    ['usagePagination', 'authQuotaPagination'].forEach(id => {
      const pagination = $(id);
      if (pagination) pagination.innerHTML = '';
    });
    const usageFilter = $('usageFilterState');
    if (usageFilter) usageFilter.textContent = t('未设置筛选');
    const catalogStatus = $('modelCatalogStatus');
    if (catalogStatus) catalogStatus.textContent = t('加载当前代理公开的模型后，可设置价格，或启用/禁用单个模型。禁用后无法调用，也不会出现在客户端模型列表中。');
    [['keysCount', '未连接'], ['modelCatalogCount', '未连接'], ['usageByKeyCount', '未连接'], ['usageByModelCount', '未连接'], ['usageRecentCount', '未连接']].forEach(([id, text]) => {
      const target = $(id);
      if (target) target.textContent = t(text);
    });
    if ($('btnResetAllKeySpend')) $('btnResetAllKeySpend').disabled = true;
  }

  function resetDataBoundFilters() {
    ['overviewKeyFilter', 'overviewAuthFilter', 'usageKeyFilter', 'usageAuthFilter', 'authQuotaNameFilter', 'authQuotaBatchConcurrency'].forEach(id => {
      const el = $(id);
      if (el) {
        el.value = '';
        delete el.dataset.selectedKeyId;
      }
    });
    [['overviewModelFilter', '全部已使用模型'], ['usageModelFilter', '全部已使用模型'], ['authQuotaProviderFilter', '全部平台'], ['priceModelPicker', '加载后选择模型']].forEach(([id, label]) => {
      const select = $(id);
      if (!select) return;
      select.innerHTML = '<option value="">'+label+'</option>';
      select.value = '';
    });
    closeKeySearch('overview');
    closeKeySearch('usage');
    closeAuthSearch('overview');
    closeAuthSearch('usage');
  }

  function clearLoadedSession() {
    state.tabLoadSeq += 1;
    state.overview = null;
    state.keys = [];
    state.authQuotas = null;
    state.authQuotaWeeks = {};
    state.authQuotaRefreshing = {};
    state.authQuotaProvider = '';
    state.authQuotaName = '';
    state.authQuotaPage = 1;
    state.authQuotaPageRefreshing = false;
    state.allKeys = [];
    state.usedAuths = [];
    state.modelPrices = {};
    state.modelCatalogError = '';
    state.availableModels = [];
    state.usagePage = 1;
    state.usageSummary = null;
    state.usageRecent = null;
    state.deleteKeyID = '';
    state.resetSpendID = '';
    state.resetSpendAll = false;
    closeKeyModal();
    closePriceModal();
    closeDeleteKeyModal();
    closeResetSpendModal();
    resetDataBoundFilters();
    renderDisconnectedOverview();
    renderDisconnectedTabStates();
  }

  function renderOverviewEChart(id, emptyText, option, handlers) {
    const target = $(id);
    disposeOverviewChart(id);
    if (!option || !window.echarts) {
      target.className = '';
      const emptyIcons = { overviewTrend: 'chart', overviewCostTrend: 'coin', overviewModelShare: 'pie', overviewModelRank: 'trophy' };
      target.innerHTML = chartEmptyState(window.echarts ? emptyText : '图表组件加载失败，请检查网络连接后刷新页面', false, emptyIcons[id]);
      return;
    }
    target.innerHTML = '';
    target.className = 'overview-echart';
    const chart = echarts.init(target, null, { renderer: 'svg' });
    chart.setOption(option);
    Object.entries(handlers || {}).forEach(([event, handler]) => chart.on(event, handler));
    state.charts[id] = chart;
    if (id === 'overviewModelShare' && window.ResizeObserver && !state.overviewShareObserver) {
      state.overviewShareObserver = new ResizeObserver(() => {
        const share = state.charts.overviewModelShare;
        if (share) share.resize();
      });
      state.overviewShareObserver.observe(target);
    }
    requestAnimationFrame(resizeOverviewCharts);
  }

  function resizeOverviewCharts() {
    Object.values(state.charts).forEach(chart => chart.resize());
  }

  function getTrendGrain(chart) {
    const grain = chart === 'cost' ? state.costTrendGrain : state.tokenTrendGrain;
    return grain === 'hour' || grain === 'month' ? grain : 'day';
  }

  function trendGrainUnit(grain) {
    if (grain === 'hour') return '小时';
    if (grain === 'month') return '自然月';
    return '自然日';
  }

  function overviewTrendBucket(date, grain) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hour = String(date.getHours()).padStart(2, '0');
    if (grain === 'hour') return { key: year + '-' + month + '-' + day + 'T' + hour, label: month + '-' + day + ' ' + hour + ':00' };
    if (grain === 'month') return { key: year + '-' + month, label: year + '-' + month };
    return { key: year + '-' + month + '-' + day, label: month + '-' + day };
  }

  function overviewTrendSeries(items, grain) {
    grain = grain === 'hour' || grain === 'month' ? grain : 'day';
    const buckets = new Map();
    (items || []).forEach(item => {
      const date = item.created_at ? new Date(item.created_at) : null;
      if (!date || Number.isNaN(date.getTime())) return;
      const bucket = overviewTrendBucket(date, grain);
      const current = buckets.get(bucket.key) || {
        key: bucket.key,
        date: bucket.label,
        input: 0,
        output: 0,
        cacheRead: 0,
        cacheRelated: 0,
        tokens: 0,
        cost: 0,
      };
      current.input += Number(item.input_tokens || 0);
      current.output += Number(item.output_tokens || 0);
      current.cacheRead += Number(item.cache_read_tokens || 0);
      current.cacheRelated += usageCacheRelated(item);
      current.tokens += usageTokens(item);
      current.cost += usageCost(item);
      buckets.set(bucket.key, current);
    });
    return [...buckets.values()]
      .sort((a, b) => a.key.localeCompare(b.key))
      .map(point => ({
        ...point,
        cacheHitRate: point.cacheRelated > 0 ? point.cacheRead / point.cacheRelated * 100 : 0,
      }));
  }

  function syncTrendGrainSwitch(chart) {
    const charts = chart ? [chart] : ['token', 'cost'];
    charts.forEach(name => {
      const grain = getTrendGrain(name);
      document.querySelectorAll('.trend-grain-switch[data-trend-chart="'+name+'"] [data-trend-grain]').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.trendGrain === grain);
      });
    });
  }

  function setTrendGrain(chart, grain) {
    const next = grain === 'hour' || grain === 'month' ? grain : 'day';
    if (chart === 'cost') state.costTrendGrain = next;
    else state.tokenTrendGrain = next;
    syncTrendGrainSwitch(chart);
    if (!state.overview) return;
    const items = state.overview.recent_usage || [];
    if (chart === 'cost') renderOverviewCostTrend(items);
    else renderOverviewTrend(items);
  }

  function trendMetric(label, value, color, title, series, dashed) {
    return '<button type="button" class="trend-metric'+(dashed?' dashed':'')+'" data-series="'+esc(series)+'"'+(title?' title="'+esc(title)+'"':'')+'><i style="background:'+color+'"></i>'+esc(t(label))+' <b>'+esc(value)+'</b></button>';
  }

  function bindTrendMetricToggles(rowID, chartID) {
    const row = $(rowID);
    if (!row || !state.charts[chartID]) return;
    row.querySelectorAll('[data-series]').forEach(chip => {
      chip.addEventListener('click', () => {
        const chart = state.charts[chartID];
        const name = chip.dataset.series;
        if (!chart || !name) return;
        chart.dispatchAction({ type: 'legendToggleSelect', name });
        chip.classList.toggle('off');
      });
    });
  }

  function renderOverviewLineChart(options) {
    const {
      targetID,
      points,
      series,
      emptyText,
    } = options;
    if (!points.length) {
      renderOverviewEChart(targetID, emptyText, null);
      return;
    }
    const hasRightAxis = series.some(item => item.axis === 'right');
    const sparse = points.length <= 6;
    renderOverviewEChart(targetID, emptyText, {
      color: series.map(item => item.color),
      animationDuration: 420,
      animationEasing: 'cubicOut',
      grid: { top: 12, right: hasRightAxis ? 8 : 8, bottom: points.length > 10 ? 36 : 8, left: 8, containLabel: true },
      legend: { show: false, data: series.map(item => item.label) },
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(26, 31, 28, .94)',
        borderWidth: 0,
        padding: [10, 12],
        textStyle: { color: '#fff', fontSize: 12 },
        axisPointer: { type: 'line', lineStyle: { color: '#9ea9a1', type: 'dashed' } },
        formatter(params) {
          const rows = params.map(param => {
            const item = series[param.seriesIndex];
            return param.marker + item.label + '<span style="float:right;margin-left:20px;font-weight:700">' + item.format(param.value, points[param.dataIndex]) + '</span>';
          });
          return '<div style="min-width:150px"><strong>'+esc(params[0].axisValue)+'</strong><div style="height:6px"></div>' + rows.join('<br/>') + '</div>';
        },
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: points.map(point => point.date),
        axisLine: { lineStyle: { color: '#dce5de' } },
        axisTick: { show: false },
        axisLabel: { color: '#6a766e', fontSize: 11, hideOverlap: true, margin: 10 },
      },
      yAxis: [
        {
          type: 'value',
          min: 0,
          splitNumber: 3,
          axisLabel: { color: '#6a766e', fontSize: 11, formatter: value => options.formatLeft(value) },
          splitLine: { lineStyle: { color: '#edf1ee', type: 'dashed' } },
          axisLine: { show: false },
          axisTick: { show: false },
        },
        ...(hasRightAxis ? [{
          type: 'value', min: 0, max: 100, splitNumber: 2,
          axisLabel: { color: '#c56b82', fontSize: 11, formatter: value => options.formatRight(value) },
          splitLine: { show: false }, axisLine: { show: false }, axisTick: { show: false },
        }] : []),
      ],
      dataZoom: points.length > 10 ? [{
        type: 'inside', start: 0, end: 100, zoomOnMouseWheel: 'shift', moveOnMouseMove: true,
      }, {
        type: 'slider', height: 14, bottom: 2, start: 0, end: 100,
        borderColor: 'transparent', backgroundColor: '#f0f4f1', fillerColor: 'rgba(30,183,135,.20)',
        handleStyle: { color: '#1eb787', borderColor: '#1eb787' }, textStyle: { color: 'transparent' },
      }] : [],
      series: series.map(item => ({
        name: item.label,
        type: 'line',
        yAxisIndex: item.axis === 'right' ? 1 : 0,
        z: item.dashed ? 1 : 3,
        smooth: sparse ? false : 0.35,
        showSymbol: points.length <= 24,
        symbol: 'circle',
        symbolSize: item.dashed ? 4 : 6,
        lineStyle: { width: item.dashed ? 1.6 : 2.4, type: item.dashed ? 'dashed' : 'solid', opacity: item.dashed ? 0.8 : 1 },
        itemStyle: { borderWidth: 2, borderColor: '#fff' },
        areaStyle: item.fill ? { color: item.fill } : undefined,
        emphasis: { focus: 'series', scale: true },
        data: points.map(point => Number(point[item.key] || 0)),
      })),
      media: [{
        query: { maxWidth: 560 },
        option: {
          grid: { top: 10, right: 4, bottom: points.length > 10 ? 34 : 6, left: 4, containLabel: true },
          xAxis: { axisLabel: { fontSize: 10 } },
          yAxis: [
            { axisLabel: { fontSize: 10, formatter: value => options.formatLeft(value) } },
            ...(hasRightAxis ? [{ axisLabel: { fontSize: 10, color: '#c56b82', formatter: value => options.formatRight(value) } }] : []),
          ],
        },
      }],
    });
  }

  function renderOverviewTrend(items) {
    const grain = getTrendGrain('token');
    const points = overviewTrendSeries(items, grain);
    const totalInput = points.reduce((sum, point) => sum + point.input, 0);
    const totalOutput = points.reduce((sum, point) => sum + point.output, 0);
    const totalCacheRead = points.reduce((sum, point) => sum + point.cacheRead, 0);
    const totalCacheRelated = points.reduce((sum, point) => sum + point.cacheRelated, 0);
    const avgHit = totalCacheRelated > 0 ? totalCacheRead / totalCacheRelated * 100 : 0;
    $('overviewTrendTotal').innerHTML = points.length
      ? trendMetric('入', formatTokens(totalInput), '#1eb787', tokenTitle(totalInput), '输入') +
        trendMetric('出', formatTokens(totalOutput), '#7968ee', tokenTitle(totalOutput), '输出') +
        trendMetric('缓存', formatTokens(totalCacheRead), '#f0a04b', tokenTitle(totalCacheRead), '缓存读取') +
        (totalCacheRelated > 0 ? trendMetric('命中', avgHit.toFixed(1) + '%', '#e56b8a', '', '缓存命中率', true) : '')
      : '';
    $('overviewTrendHint').textContent = points.length
      ? points.length + ' 个有使用记录的' + trendGrainUnit(grain)
      : '输入 / 输出 / 缓存读取 / 缓存命中率';
    syncTrendGrainSwitch();
    renderOverviewLineChart({
      targetID: 'overviewTrend',
      points,
      emptyText: '当前筛选条件下暂无 Token 趋势',
      formatLeft: value => formatTokens(value),
      formatRight: value => value.toFixed(0) + '%',
      series: [
        {
          key: 'input',
          label: '输入',
          color: '#1eb787',
          format: value => formatTokens(value) + ' Token',
        },
        {
          key: 'output',
          label: '输出',
          color: '#7968ee',
          format: value => formatTokens(value) + ' Token',
        },
        {
          key: 'cacheRead',
          label: '缓存读取',
          color: '#f0a04b',
          format: value => formatTokens(value) + ' Token',
        },
        {
          key: 'cacheHitRate',
          label: '缓存命中率',
          color: '#e56b8a',
          axis: 'right',
          dashed: true,
          format: value => value.toFixed(1) + '%',
        },
      ],
    });
    bindTrendMetricToggles('overviewTrendTotal', 'overviewTrend');
  }

  function renderOverviewCostTrend(items) {
    const grain = getTrendGrain('cost');
    const points = overviewTrendSeries(items, grain);
    const total = points.reduce((sum, point) => sum + point.cost, 0);
    $('overviewCostTrendTotal').innerHTML = points.length ? trendMetric('费用', formatMoney(total), '#1eb787', '', '费用') : '';
    $('overviewCostTrendHint').textContent = points.length
      ? points.length + ' 个有使用记录的' + trendGrainUnit(grain)
      : '按' + trendGrainUnit(grain) + '汇总实际费用';
    syncTrendGrainSwitch();
    renderOverviewLineChart({
      targetID: 'overviewCostTrend',
      points,
      emptyText: '当前筛选条件下暂无费用趋势',
      formatLeft: value => formatMoney(value),
      series: [
        {
          key: 'cost',
          label: '费用',
          color: '#1eb787',
          fill: 'rgba(30,183,135,.13)',
          format: value => formatMoney(value),
        },
      ],
    });
    bindTrendMetricToggles('overviewCostTrendTotal', 'overviewCostTrend');
  }

  function renderOverviewModelShareLegend(models, metricTotal) {
    const legend = $('overviewModelShareLegend');
    if (!legend) return;
    if (!models.length) {
      legend.hidden = true;
      legend.innerHTML = '';
      return;
    }
    legend.hidden = false;
    legend.innerHTML = models.map((item, index) => {
      const value = modelShareMetricValue(item, getModelShareMetric());
      const percent = metricTotal ? value / metricTotal * 100 : 0;
      const color = MODEL_SHARE_COLORS[index % MODEL_SHARE_COLORS.length];
      return '<button type="button" class="share-legend-item" data-model="'+esc(item.model)+'">' +
        '<i class="share-legend-dot" style="background:'+color+'"></i>' +
        '<span class="share-legend-body">' +
          '<span class="share-legend-top">' +
            '<span class="share-legend-name">'+esc(item.model)+'</span>' +
            '<span class="share-legend-pct">'+percent.toFixed(1)+'%</span>' +
          '</span>' +
          '<span class="share-legend-metrics">' +
            '<span class="share-legend-metric"><em>调用次数</em><b>'+esc(item.requests.toLocaleString())+'</b></span>' +
            '<span class="share-legend-metric"><em>Token</em><b>'+esc(formatTokens(item.tokens))+'</b></span>' +
            '<span class="share-legend-metric"><em>费用</em><b title="'+esc(moneyTitle(item.cost))+'">'+esc(formatMoney(item.cost))+'</b></span>' +
          '</span>' +
        '</span>' +
      '</button>';
    }).join('');
    legend.querySelectorAll('.share-legend-item').forEach((btn, index) => {
      btn.addEventListener('click', () => drillOverviewModel(btn.dataset.model));
      btn.addEventListener('mouseenter', () => {
        const chart = state.charts.overviewModelShare;
        if (chart) chart.dispatchAction({ type: 'highlight', seriesIndex: 0, dataIndex: index });
      });
      btn.addEventListener('mouseleave', () => {
        const chart = state.charts.overviewModelShare;
        if (chart) chart.dispatchAction({ type: 'downplay', seriesIndex: 0, dataIndex: index });
      });
    });
  }

  function renderOverviewModels(items) {
    const metric = getModelShareMetric();
    const models = overviewModelUsage(items).sort((a, b) =>
      modelShareMetricValue(b, metric) - modelShareMetricValue(a, metric) ||
      b.requests - a.requests ||
      a.model.localeCompare(b.model)
    );
    $('overviewModelCount').textContent = models.length ? models.length + ' 个模型' : '';
    $('overviewModelCount').hidden = !models.length;
    syncModelShareMetricSwitch();
    if (!models.length) {
      renderOverviewModelShareLegend([]);
      renderOverviewEChart('overviewModelShare', '当前筛选条件下暂无模型使用数据', null);
      return;
    }
    const metricTotal = models.reduce((sum, item) => sum + modelShareMetricValue(item, metric), 0);
    renderOverviewModelShareLegend(models, metricTotal);
    renderOverviewEChart('overviewModelShare', '当前筛选条件下暂无模型使用数据', {
      color: models.map((_, index) => MODEL_SHARE_COLORS[index % MODEL_SHARE_COLORS.length]),
      animationDuration: 420,
      animationEasing: 'cubicOut',
      legend: { show: false },
      tooltip: {
        trigger: 'item',
        backgroundColor: 'rgba(26, 31, 28, .94)',
        borderWidth: 0,
        padding: [10, 12],
        textStyle: { color: '#fff', fontSize: 12 },
        formatter(param) {
          const item = param.data.item;
          return '<strong>'+esc(item.model)+'</strong><br/>'+
            '<span style="color:#cdd6cf">'+esc(t('调用次数'))+'</span><span style="float:right;margin-left:20px;font-weight:700">'+esc(item.requests.toLocaleString())+'</span><br/>'+
            '<span style="color:#cdd6cf">Token</span><span style="float:right;margin-left:20px;font-weight:700">'+esc(formatTokens(item.tokens))+'</span><br/>'+
            '<span style="color:#cdd6cf">'+esc(t('费用'))+'</span><span style="float:right;margin-left:20px;font-weight:700">'+esc(formatMoney(item.cost))+'</span><br/>'+
            '<span style="color:#cdd6cf">'+esc(t('占比'))+'</span><span style="float:right;margin-left:20px">'+param.percent.toFixed(1)+'%</span>';
        },
      },
      series: [{
        name: modelShareCenterLabel(metric),
        type: 'pie',
        radius: ['42%', '66%'],
        center: ['50%', '50%'],
        minAngle: 3,
        avoidLabelOverlap: true,
        itemStyle: { borderColor: '#fff', borderWidth: 3, borderRadius: 4 },
        label: {
          show: true,
          position: 'center',
          formatter: '{total|' + formatModelShareCenter(metricTotal, metric) + '}\n{caption|' + modelShareCenterLabel(metric) + '}',
          rich: {
            total: { color: '#1f2922', fontSize: 18, fontWeight: 700, lineHeight: 24 },
            caption: { color: '#5f6a61', fontSize: 11, fontWeight: 600, lineHeight: 16 },
          },
        },
        labelLine: { show: false },
        emphasis: { scale: true, scaleSize: 7, itemStyle: { shadowBlur: 12, shadowColor: 'rgba(31,41,34,.18)' } },
        data: models.map(item => ({ name: item.model, value: modelShareMetricValue(item, metric), item })),
      }],
    }, {
      click: params => {
        if (params.componentType === 'series' && params.data && params.data.item) drillOverviewModel(params.data.item.model);
      },
    });
  }

  function rankBarPercent(item, ranked, spec) {
    if (!spec.ready(item)) return 0;
    const score = spec.score(item);
    if (score === Infinity) return 100;
    const finite = ranked.filter(spec.ready).map(spec.score).filter(Number.isFinite);
    if (!finite.length || !Number.isFinite(score)) return 0;
    if (spec.higher) {
      const max = Math.max(...finite);
      return max > 0 ? Math.max(8, score / max * 100) : 0;
    }
    const min = Math.min(...finite.filter(value => value > 0));
    if (!Number.isFinite(min) || score <= 0) return score <= 0 ? 100 : 0;
    return Math.max(8, min / score * 100);
  }

  function renderOverviewRank(items) {
    const spec = modelRankSpec();
    const models = overviewModelUsage(items).sort((a, b) => {
      const aReady = spec.ready(a);
      const bReady = spec.ready(b);
      if (aReady !== bReady) return aReady ? -1 : 1;
      if (!aReady) return a.model.localeCompare(b.model);
      const aScore = spec.score(a);
      const bScore = spec.score(b);
      if (aScore === bScore) return b.requests - a.requests || a.model.localeCompare(b.model);
      if (aScore === Infinity) return -1;
      if (bScore === Infinity) return 1;
      if (!Number.isFinite(aScore)) return 1;
      if (!Number.isFinite(bScore)) return -1;
      return spec.higher ? (bScore - aScore) : (aScore - bScore);
    });
    const target = $('overviewModelRank');
    const count = $('overviewRankCount');
    const hint = $('overviewRankHint');
    syncModelRankMetricSwitch();
    if (hint) hint.textContent = t(spec.hint);
    if (count) { count.textContent = models.length ? models.length + ' 个模型' : ''; count.hidden = !models.length; }
    if (!target) return;
    if (!models.length) {
      target.innerHTML = chartEmptyState(t('当前筛选条件下暂无模型效率数据'), false, 'trophy');
      return;
    }
    target.innerHTML = models.map((item, index) => {
      const ready = spec.ready(item);
      const score = ready ? spec.score(item) : NaN;
      const value = ready && (Number.isFinite(score) || score === Infinity) ? spec.format(score) : '—';
      const width = rankBarPercent(item, models, spec);
      const title = ['value', 'unit', 'cache', 'tps'].map(key => {
        const metric = modelRankSpec(key);
        const metricScore = metric.ready(item) ? metric.score(item) : NaN;
        const metricValue = metric.ready(item) && (Number.isFinite(metricScore) || metricScore === Infinity)
          ? metric.format(metricScore) : '—';
        return t(metric.label) + ' ' + metricValue;
      }).join('\n');
      return '<button type="button" class="model-rank-item" data-model="'+esc(item.model)+'" title="'+esc(title)+'">' +
        '<span class="model-rank-index">'+(index + 1)+'</span>' +
        '<span class="model-rank-body">' +
          '<span class="model-rank-top">' +
            '<span class="model-rank-name">'+esc(item.model)+'</span>' +
            '<span class="model-rank-value">'+esc(value)+'</span>' +
          '</span>' +
          '<span class="model-rank-bar" aria-hidden="true"><i style="width:'+width.toFixed(1)+'%"></i></span>' +
          '<span class="model-rank-meta">'+esc(item.requests.toLocaleString() + ' ' + t('次') + ' · ' + formatTokens(item.tokens) + ' Token · ' + formatMoney(item.cost))+'</span>' +
        '</span>' +
      '</button>';
    }).join('');
    target.querySelectorAll('.model-rank-item').forEach(btn => {
      btn.addEventListener('click', () => drillOverviewModel(btn.dataset.model));
    });
  }

  function renderOverviewModelFilter(items) {
    const select = $('overviewModelFilter');
    const current = select.value;
    const models = [...new Set((items || []).map(item => {
      const value = item.Model != null ? item.Model : item.model;
      return String(value || '').trim();
    }).filter(Boolean))].sort((a, b) => a.localeCompare(b));
    select.innerHTML = '<option value="">全部已使用模型</option>' + models.map(model =>
      '<option value="'+esc(model)+'">'+esc(model)+'</option>'
    ).join('');
    if (models.includes(current)) select.value = current;
    refreshCustomControl(select);
  }

  function renderOverview(data) {
    const keys = (data.keys || []).filter(key => !key.revoked_at);
    const items = data.recent_usage || [];
    state.usedAuths = data.used_auths || [];
    renderOverviewModelFilter(data.used_models || []);
    renderAuthSearchOptions('overview');
    renderAuthSearchOptions('usage');
    const activeKeys = keys.filter(k => k.enabled && !k.revoked_at).length;
    const totalTokens = items.reduce((sum, item) => sum + usageTokens(item), 0);
    const totalSpend = items.reduce((sum, item) => sum + Number(item.cost_micro_usd || 0), 0);
    const totalReq = items.length;
    const filter = getOverviewFilterValues();
    const rangeText = $('overviewRangeFilter').selectedOptions[0].textContent;
    const labels = [rangeText];
    if (filter.plugin_key_id) labels.push('指定密钥');
    if (filter.auth_id || filter.auth_index) labels.push('账号');
    if (filter.model) labels.push('模型: ' + filter.model);
    if (filter.source) labels.push('来源: ' + filter.source);
    $('overviewFilterState').textContent = labels.join(' · ');
    $('overviewStats').innerHTML = [
      ['key', '密钥数 / 可用', keys.length + ' / ' + activeKeys],
      ['activity', '筛选请求', totalReq.toLocaleString()],
      ['layers', '筛选 Token', formatTokens(totalTokens)],
      ['coin', '筛选费用', formatMoney(totalSpend)],
      ['cpu', '模型数量', overviewModelUsage(items).length.toLocaleString()],
    ].map(([icon, k, v]) => '<div class="stat"><div class="k"><span class="stat-icon" aria-hidden="true">'+uiIcon(icon)+'</span>'+esc(k)+'</div><div class="v">'+esc(v)+'</div></div>').join('');
    renderOverviewTrend(items);
    renderOverviewCostTrend(items);
    renderOverviewModels(items);
    renderOverviewRank(items);
    requestAnimationFrame(resizeOverviewCharts);
  }

  function renderKeys(keys) {
    const allKeys = [...(keys || [])].sort((left, right) => {
      if (Boolean(left.revoked_at) !== Boolean(right.revoked_at)) return left.revoked_at ? 1 : -1;
      return String(right.created_at || '').localeCompare(String(left.created_at || ''));
    });
    // Keep revoked keys available for historical usage filters, but hide them
    // from the active Key management table.
    state.keys = allKeys.filter(key => !key.revoked_at);
    state.allKeys = allKeys;
    renderKeySearchOptions('usage');
    renderKeySearchOptions('overview');

    if ($('keysCount')) {
      $('keysCount').textContent = state.keys.length ? (state.keys.length + ' ' + t('个密钥')) : '暂无密钥';
    }
    if ($('btnResetAllKeySpend')) $('btnResetAllKeySpend').disabled = !state.keys.length;

    if (!state.keys.length) {
      $('keysTable').innerHTML = '<div class="keys-empty empty-state"><span class="empty-icon" aria-hidden="true">'+uiIcon('key')+'</span><span>还没有密钥。点击右上角“添加密钥”创建第一个额度凭证。</span></div>';
      return;
    }

    const money = (v) => formatMoney(v);

    const tokenLimitLabel = (period) => {
      const tokens = Number(period && period.tokens || 0);
      if (tokens > 0) return formatTokens(tokens);
      return (period && period.mode) === 'available' ? t('可用') : t('无限制');
    };
    const modelChips = (models, tokenLimits, unmatchedMode) => {
      const chips = (!models || !models.length)
        ? '<span class="model-chip all">全部模型</span>'
        : models.slice(0, 3).map(m => '<span class="model-chip" title="'+esc(m)+'">'+esc(m)+'</span>').join('') + (models.length > 3 ? '<span class="model-chip">+'+ (models.length - 3) +'</span>' : '');
      const limits = tokenLimits || [];
      const unmatched = unmatchedMode === 'disabled' ? t('未匹配禁用') : t('未匹配可用');
      const limitChip = '<span class="token-limit-chip'+(unmatchedMode === 'disabled' ? ' warn' : '')+'" title="'+esc((limits.length ? limits.map(item => item.model+' '+t('日')+' '+tokenLimitLabel(item.daily)+' / '+t('周')+' '+tokenLimitLabel(item.weekly)+' / '+t('月')+' '+tokenLimitLabel(item.monthly)).join('\n')+'\n' : '') + unmatched)+'">'+esc((limits.length ? limits.length + ' ' + t('模型 Token 限制') + ' · ' : '') + unmatched)+'</span>';
      return '<div>' + '<div class="model-chip-row">' + chips + '</div>' + limitChip + '</div>';
    };

    const limitText = (value) => Number(value || 0) > 0 ? money(value) : t('不限制');
    const quotaBlock = (k) => {
      const quota = Number(k.quota_micro_usd || 0);
      const used = Number(k.settled_spend_micro_usd || 0);
      const periodLimits = '<span class="quota-periods">' +
        '<span><span>日</span> '+esc(limitText(k.daily_quota_micro_usd))+'</span>' +
        '<span><span>周</span> '+esc(limitText(k.weekly_quota_micro_usd))+'</span>' +
        '<span><span>月</span> '+esc(limitText(k.monthly_quota_micro_usd))+'</span>' +
        '<span><span>并发</span> '+esc(Number(k.max_concurrent_requests || 0) || t('不限制'))+'</span>' +
        '</span>';
      if (quota <= 0) {
        return '<div class="quota-cell">' +
          '<div class="quota-line"><strong>不限制</strong><span class="muted">限额</span></div>' +
          '<div class="quota-bar"><span style="width:0%"></span></div>' + periodLimits +
          '</div>';
      }
      const pct = Math.min(100, Math.max(0, (used / quota) * 100));
      const tone = pct >= 90 ? 'danger' : (pct >= 70 ? 'warn' : '');
      return '<div class="quota-cell">' +
        '<div class="quota-line"><strong>'+esc(money(quota))+'</strong><span class="muted">限额</span></div>' +
        '<div class="quota-bar '+tone+'"><span style="width:'+pct.toFixed(1)+'%"></span></div>' + periodLimits +
        '</div>';
    };

    $('keysTable').innerHTML = '<div class="table-scroll"><table class="keys-table"><thead><tr><th>标签</th><th>可用模型</th><th>密钥限额</th><th>已用 / 剩余</th><th>状态</th><th>操作</th></tr></thead><tbody>' +
      state.keys.map(k => {
        const st = k.revoked_at
          ? '<span class="badge bad">已删除</span>'
          : '<label class="key-switch" title="'+esc(k.enabled ? t('启用') : t('禁用'))+'"><input type="checkbox" role="switch" data-enable-key="'+esc(k.id)+'" aria-label="'+esc(t('启用'))+'"'+(k.enabled ? ' checked' : '')+'/><span class="key-switch-ui" aria-hidden="true"></span></label>';
        return '<tr>' +
          '<td><div class="key-label"><strong title="'+esc(k.label||'(无标签)')+'">'+esc(k.label||'(无标签)')+'</strong></div></td>' +
           '<td>'+modelChips(k.allowed_models, k.model_token_limits, k.unmatched_models_mode)+'</td>' +
          '<td>'+quotaBlock(k)+'</td>' +
          '<td><div class="spend-cell"><span class="primary">'+esc(money(k.settled_spend_micro_usd))+'</span><span class="secondary"><span>剩余</span> '+(Number(k.quota_micro_usd||0) <= 0 ? t('不限制') : esc(money(k.remaining_micro_usd)))+'</span></div></td>' +
          '<td>'+st+'</td>' +
          '<td><div class="row-actions">' +
            '<button class="btn soft sm" data-copy="'+esc(k.id)+'">'+uiIcon('copy')+esc(t('复制密钥'))+'</button>' +
            '<button class="btn primary sm" data-manage="'+esc(k.id)+'">'+uiIcon('gear')+esc(t('管理密钥'))+'</button>' +
            '<button class="btn soft sm" data-reset-spend="'+esc(k.id)+'">'+uiIcon('refresh')+esc(t('重置已用'))+'</button>' +
            '<button class="btn danger sm" data-delete="'+esc(k.id)+'">'+uiIcon('trash')+esc(t('删除'))+'</button>' +
          '</div></td></tr>';
      }).join('') + '</tbody></table></div>';

    $('keysTable').querySelectorAll('[data-copy]').forEach(btn => btn.addEventListener('click', () => copyKeyByID(btn.dataset.copy)));
    $('keysTable').querySelectorAll('[data-manage]').forEach(btn => btn.addEventListener('click', () => openKeyModal('manage', btn.dataset.manage)));
    $('keysTable').querySelectorAll('[data-reset-spend]').forEach(btn => btn.addEventListener('click', () => openResetSpendModal(btn.dataset.resetSpend)));
    $('keysTable').querySelectorAll('[data-delete]').forEach(btn => btn.addEventListener('click', () => openDeleteKeyModal(btn.dataset.delete)));
    $('keysTable').querySelectorAll('[data-enable-key]').forEach(input => input.addEventListener('change', () => toggleKeyEnabled(input.dataset.enableKey, input.checked, input)));
  }

  async function toggleKeyEnabled(id, enabled, control) {
    const wrap = control.closest('.key-switch');
    if (wrap) wrap.classList.add('is-busy');
    control.disabled = true;
    try {
      const result = await api('POST', 'credit-manager/keys/update', { id, enabled });
      [state.keys, state.allKeys].forEach(list => {
        const item = (list || []).find(key => key.id === id);
        if (item) item.enabled = result.enabled;
      });
      wrap && wrap.setAttribute('title', result.enabled ? t('启用') : t('禁用'));
      flash(result.enabled ? t('密钥已启用') : t('密钥已禁用'), true);
    } catch (e) {
      control.checked = !enabled;
      flash(e.message, false);
    } finally {
      control.disabled = false;
      wrap && wrap.classList.remove('is-busy');
    }
  }

  function keyFilterDisplayLabel(key) {
    return (key.label || '(无标签)') + (key.revoked_at ? '（已删除）' : '');
  }

  function keyFilterLabel(key) {
    return keyFilterDisplayLabel(key) + ' · ' + key.id;
  }

  function keySearchMatches(query) {
    query = String(query || '').trim().toLocaleLowerCase();
    if (!query) return state.allKeys;
    return state.allKeys.filter(key => key.id.toLocaleLowerCase().includes(query) || keyFilterLabel(key).toLocaleLowerCase().includes(query));
  }

  function renderKeySearchOptions(kind) {
    const input = $(kind + 'KeyFilter');
    const panel = $(kind + 'KeyOptions');
    const matches = keySearchMatches(input.value);
    panel.innerHTML = matches.length ? matches.map(key =>
      '<button class="key-search-option'+(key.revoked_at ? ' revoked' : '')+'" type="button" data-key-id="'+esc(key.id)+'" title="'+esc(key.id)+'">' +
        '<span class="key-search-label">'+esc(key.label || '(无标签)')+'</span>' +
      '</button>'
    ).join('') : '<div class="key-search-empty">未找到匹配的密钥</div>';
    panel.querySelectorAll('[data-key-id]').forEach(button => button.addEventListener('click', () => {
      const key = state.allKeys.find(item => item.id === button.dataset.keyId);
      if (!key) return;
      input.value = keyFilterDisplayLabel(key);
      input.dataset.selectedKeyId = key.id;
      closeKeySearch(kind);
    }));
  }

  function openKeySearch(kind) {
    closeKeySearch(kind === 'overview' ? 'usage' : 'overview');
    closeAuthSearch('overview');
    closeAuthSearch('usage');
    renderKeySearchOptions(kind);
    const wrapper = $(kind + 'KeySearch');
    const panel = $(kind + 'KeyOptions');
    wrapper.classList.add('open');
    panel.hidden = false;
  }

  function closeKeySearch(kind) {
    const wrapper = $(kind + 'KeySearch');
    const panel = $(kind + 'KeyOptions');
    if (!wrapper || !panel) return;
    wrapper.classList.remove('open');
    panel.hidden = true;
  }

  function resolveKeyFilter(control) {
    const input = typeof control === 'string' ? null : control;
    const raw = String(input ? input.value : control || '').trim();
    if (!raw) return '';
    const selectedKeyID = input && input.dataset.selectedKeyId;
    if (selectedKeyID && state.allKeys.some(key => key.id === selectedKeyID)) return selectedKeyID;
    const needle = raw.toLocaleLowerCase();
    const matches = state.allKeys.filter(key =>
      key.id.toLocaleLowerCase().includes(needle) || keyFilterLabel(key).toLocaleLowerCase().includes(needle)
    );
    if (matches.length === 1) return matches[0].id;
    if (matches.length > 1) throw new Error('匹配多个密钥，请从搜索建议中选择完整项');
    throw new Error('未找到匹配的密钥');
  }

  function authFilterValue(auth, key) {
    if (!auth) return '';
    const camel = key.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
    for (const name of [key, camel]) if (auth[name] != null) return String(auth[name] || '').trim();
    return '';
  }

  function formatProviderDisplay(provider) {
    const value = String(provider || '').trim();
    if (!value) return '';
    const key = value.toLowerCase();
    if (key === 'xai' || key === 'grok') return 'xAI';
    if (key === 'codex' || key === 'openai') return 'Codex';
    if (key === 'claude' || key === 'anthropic') return 'Claude';
    if (key === 'antigravity' || key === 'google' || key === 'gemini') return 'Antigravity';
    if (key === 'kimi' || key === 'moonshot') return 'Kimi';
    const prefixes = [
      'openai-compatible-', 'openai-compatibility-', 'openai-compatible/',
      'gemini-api-key-', 'claude-api-key-', 'codex-api-key-',
      'xai-api-key-', 'vertex-api-key-', 'interactions-api-key-',
    ];
    for (const prefix of prefixes) {
      if (key.startsWith(prefix)) {
        const rest = value.slice(prefix.length).replace(/\.(json|ya?ml)$/i, '').trim();
        return rest || value;
      }
    }
    return value.replace(/\.(json|ya?ml)$/i, '');
  }

  function sameAuthText(a, b) {
    const left = String(a || '').trim().toLowerCase();
    const right = String(b || '').trim().toLowerCase();
    return !!left && left === right;
  }

  function formatAuthAccount(account, provider, providerDisplay) {
    let value = String(account || '').trim().replace(/\.(json|ya?ml)$/i, '');
    if (!value || value === '未知账号') return '';
    const pretty = providerDisplay || formatProviderDisplay(provider);
    const accountPretty = formatProviderDisplay(value);
    if (sameAuthText(value, provider) || sameAuthText(value, pretty) || sameAuthText(accountPretty, pretty)) return '';
    return accountPretty && accountPretty !== value ? accountPretty : value;
  }

  function formatAuthDisplay(provider, account) {
    const providerDisplay = formatProviderDisplay(provider);
    const accountDisplay = formatAuthAccount(account, provider, providerDisplay);
    return [providerDisplay, accountDisplay].filter(Boolean).join(' · ');
  }

  function authFilterLabel(auth) {
    const provider = authFilterValue(auth, 'auth_provider') || authFilterValue(auth, 'provider');
    const account = authFilterValue(auth, 'auth_label') || authFilterValue(auth, 'label')
      || authFilterValue(auth, 'auth_email') || authFilterValue(auth, 'email')
      || authFilterValue(auth, 'auth_name') || authFilterValue(auth, 'name')
      || authFilterValue(auth, 'auth_id') || authFilterValue(auth, 'auth_index');
    return formatAuthDisplay(provider, account) || t('未知账号');
  }

  function authFilterKey(auth) {
    return [authFilterValue(auth, 'auth_provider') || authFilterValue(auth, 'provider'),
      authFilterValue(auth, 'auth_id'), authFilterValue(auth, 'auth_index')].join('\t');
  }

  function authSearchMatches(query) {
    query = String(query || '').trim().toLocaleLowerCase();
    const items = state.usedAuths || [];
    if (!query) return items;
    return items.filter(auth => {
      const haystack = [
        authFilterLabel(auth),
        authFilterValue(auth, 'auth_id'),
        authFilterValue(auth, 'auth_index'),
        authFilterValue(auth, 'auth_provider') || authFilterValue(auth, 'provider'),
        authFilterValue(auth, 'auth_label') || authFilterValue(auth, 'label'),
        authFilterValue(auth, 'auth_email') || authFilterValue(auth, 'email'),
        authFilterValue(auth, 'auth_name') || authFilterValue(auth, 'name'),
      ].join(' ').toLocaleLowerCase();
      return haystack.includes(query);
    });
  }

  function renderAuthSearchOptions(kind) {
    const input = $(kind + 'AuthFilter');
    const panel = $(kind + 'AuthOptions');
    if (!input || !panel) return;
    const matches = authSearchMatches(input.value);
    panel.innerHTML = matches.length ? matches.map((auth, index) =>
      '<button class="key-search-option" type="button" data-auth-pos="'+index+'" title="'+esc(authFilterValue(auth, 'auth_id') || authFilterValue(auth, 'auth_index'))+'">' +
        '<span class="key-search-label">'+esc(authFilterLabel(auth))+'</span>' +
      '</button>'
    ).join('') : '<div class="key-search-empty">未找到匹配的账号</div>';
    panel.querySelectorAll('[data-auth-pos]').forEach(button => button.addEventListener('mousedown', event => {
      event.preventDefault();
      const auth = matches[Number(button.dataset.authPos)];
      if (!auth) return;
      input.value = authFilterLabel(auth);
      closeAuthSearch(kind);
    }));
  }

  function openAuthSearch(kind) {
    closeAuthSearch(kind === 'overview' ? 'usage' : 'overview');
    closeKeySearch('overview');
    closeKeySearch('usage');
    renderAuthSearchOptions(kind);
    const wrapper = $(kind + 'AuthSearch');
    const panel = $(kind + 'AuthOptions');
    if (!wrapper || !panel) return;
    wrapper.classList.add('open');
    panel.hidden = false;
  }

  function closeAuthSearch(kind) {
    const wrapper = $(kind + 'AuthSearch');
    const panel = $(kind + 'AuthOptions');
    if (!wrapper || !panel) return;
    wrapper.classList.remove('open');
    panel.hidden = true;
  }

  function resolveAuthFilter(raw) {
    raw = String(raw || '').trim();
    if (!raw) return { auth_id: '', auth_provider: '', auth_index: '' };
    const items = state.usedAuths || [];
    const exact = items.find(auth => authFilterLabel(auth) === raw
      || authFilterValue(auth, 'auth_id') === raw
      || authFilterValue(auth, 'auth_index') === raw);
    const pick = auth => ({
      auth_id: authFilterValue(auth, 'auth_id'),
      auth_provider: authFilterValue(auth, 'auth_provider') || authFilterValue(auth, 'provider'),
      auth_index: authFilterValue(auth, 'auth_index'),
    });
    if (exact) return pick(exact);
    const matches = authSearchMatches(raw);
    if (matches.length === 1) return pick(matches[0]);
    if (matches.length > 1) throw new Error('匹配多个账号，请从搜索建议中选择完整项');
    throw new Error('未找到匹配的账号');
  }

  async function copyText(text) {
    const value = String(text || '');
    if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
      try {
        await navigator.clipboard.writeText(value);
        return;
      } catch (_) {
        // Some mobile browsers expose the API but deny clipboard permission.
      }
    }
    const textarea = document.createElement('textarea');
    textarea.value = value;
    textarea.setAttribute('readonly', '');
    textarea.style.cssText = 'position:fixed;left:-9999px;top:0;opacity:0';
    document.body.appendChild(textarea);
    textarea.select();
    textarea.setSelectionRange(0, value.length);
    const copied = document.execCommand && document.execCommand('copy');
    textarea.remove();
    if (!copied) throw new Error('复制失败，请手动复制');
  }

  async function copyKeyByID(id) {
    try {
      const result = await api('POST', 'credit-manager/keys/reveal', { id });
      await copyText(result.plaintext || '');
      flash('已复制密钥', true);
    } catch (e) { flash(e.message || '复制失败', false); }
  }

  async function revealKey(id) {
    try {
      const result = await api('POST', 'credit-manager/keys/reveal', { id });
      const key = state.keys.find(item => item.id === id);
      $('keyPlaintext').textContent = result.plaintext || '';
      $('keyPlainResult').classList.remove('hidden');
      $('keyModalHint').textContent = '正在查看「' + (key && key.label || '密钥') + '」的明文凭据；可复制后继续管理。';
      flash('已读取加密保存的密钥明文', true);
    } catch (e) { flash(e.message, false); }
  }

  function createKeyMaterialSuffix() {
    const bytes = new Uint8Array(42);
    crypto.getRandomValues(bytes);
    const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
    const encoded = Array.from(bytes, byte => alphabet[byte & 31]).join('');
    return encoded.slice(0, 16) + '-' + encoded.slice(16);
  }

  function keyMaterialFromInput() {
    const suffix = $('keyMaterial').value.trim().replace(/^tk-/i, '');
    return suffix ? 'tk-' + suffix : '';
  }

  function setKeyModels(models) {
    const selected = new Set((models || []).map(value => String(value).trim()).filter(Boolean));
    const rules = (state.overview && state.overview.pricing) || [];
    const available = [...new Set([...(state.availableModels || []), ...selected])].filter(modelID => selected.has(modelID) || !modelIsDisabled(modelID, rules)).sort();
    const picker = $('keyModalModels');
    picker.innerHTML = available.map(modelID =>
      '<option value="' + esc(modelID) + '"' + (selected.has(modelID) ? ' selected' : '') + '>' + esc(modelID) + '</option>'
    ).join('');
    $('keyModalModelsHint').textContent = available.length
      ? '可直接勾选多个模型；不选择任何模型表示全部模型可用。'
      : '未发现可用模型；请确认宿主管理密钥、上游认证文件或 AI 提供商。';
    setKeyTokenLimitModels(available);
    refreshCustomControl(picker);
  }

  function selectedKeyModels() {
    return Array.from($('keyModalModels').selectedOptions).map(option => option.value);
  }

  function tokenLimitPeriodValue(period) {
    period = period || {};
    const tokens = Number(period.tokens || 0);
    return {
      tokens: Number.isFinite(tokens) && tokens > 0 ? Math.floor(tokens) : 0,
      mode: period.mode === 'available' ? 'available' : 'unlimited',
    };
  }

  function unmatchedModelsMode() {
    return $('keyModalUnmatched').dataset.unmatched === 'disabled' ? 'disabled' : 'available';
  }

  function setUnmatchedModelsMode(mode, disabled) {
    mode = mode === 'disabled' ? 'disabled' : 'available';
    const wrap = $('keyModalUnmatched');
    wrap.dataset.unmatched = mode;
    wrap.classList.toggle('is-disabled', mode === 'disabled');
    wrap.querySelectorAll('[data-unmatched-set]').forEach(button => {
      button.classList.toggle('active', button.dataset.unmatchedSet === mode);
      button.disabled = Boolean(disabled);
    });
    $('keyModalUnmatchedHint').textContent = mode === 'disabled'
      ? t('仅下方列出的模型可以调用；未匹配的模型会被拒绝。')
      : t('未列入下方的模型可以调用，且不限制 Token。');
  }

  function tokenLimitsEnabled() {
    return $('keyModalTokenLimitsEnabled').checked;
  }

  function setTokenLimitsSectionVisible(on, disabled) {
    $('keyModalTokenLimitsEnabled').checked = Boolean(on);
    $('keyModalTokenLimitsEnabled').disabled = Boolean(disabled);
    $('keyModalTokenLimitsSection').classList.toggle('hidden', !on);
    if (!on) closeTokenLimitModelSearch();
  }

  function collectKeyTokenLimitPayload() {
    if (!tokenLimitsEnabled()) {
      return { model_token_limits: [], unmatched_models_mode: 'available' };
    }
    return { model_token_limits: collectModelTokenLimits(), unmatched_models_mode: unmatchedModelsMode() };
  }

  function collectModelTokenLimits() {
    return Array.from(document.querySelectorAll('#keyModalTokenLimits .token-limit-row')).map(row => {
      const read = (name) => {
        const period = row.querySelector('[data-period="'+name+'"]');
        return tokenLimitPeriodValue({
          tokens: period.querySelector('[data-tokens]').value,
          mode: period.dataset.mode,
        });
      };
      return { model: row.dataset.model, total: read('total'), daily: read('daily'), weekly: read('weekly'), monthly: read('monthly') };
    }).filter(item => item.model);
  }

  function syncTokenLimitPeriod(periodEl) {
    const input = periodEl.querySelector('input[data-tokens]');
    const mode = periodEl.dataset.mode === 'available' ? 'available' : 'unlimited';
    const capped = String(input.value || '').trim() !== '' && Number(input.value) > 0;
    periodEl.dataset.mode = mode;
    periodEl.classList.toggle('is-capped', capped);
    periodEl.querySelectorAll('[data-mode-set]').forEach(button => {
      button.classList.toggle('active', button.dataset.modeSet === mode);
      button.disabled = capped || input.disabled;
    });
  }

  function renderKeyTokenLimits(limits, disabled) {
    const items = Array.isArray(limits) ? limits : [];
    const target = $('keyModalTokenLimits');
    if (!items.length) {
      target.innerHTML = '<p class="token-limit-empty">'+esc(t('添加模型后，可设置总 / 日 / 周 / 月 Token 上限'))+'</p>';
      return;
    }
    const periodField = (name, label, period) => {
      const value = tokenLimitPeriodValue(period);
      const capped = value.tokens > 0;
      return '<div class="token-limit-period'+(capped ? ' is-capped' : '')+'" data-period="'+name+'" data-mode="'+value.mode+'"><span>'+esc(t(label))+'</span>' +
        '<input type="number" min="1" step="1" inputmode="numeric" data-tokens placeholder="'+esc(t('未填则选可用或无限制'))+'" value="'+(capped ? value.tokens : '')+'"'+(disabled ? ' disabled' : '')+'/>' +
        '<div class="token-limit-mode" role="group">' +
          '<button type="button" data-mode-set="available"'+(value.mode === 'available' ? ' class="active"' : '')+(disabled || capped ? ' disabled' : '')+'>'+esc(t('可用'))+'</button>' +
          '<button type="button" data-mode-set="unlimited"'+(value.mode !== 'available' ? ' class="active"' : '')+(disabled || capped ? ' disabled' : '')+'>'+esc(t('无限制'))+'</button>' +
        '</div></div>';
    };
    target.innerHTML = items.map(item => {
      const model = String(item.model || '').trim();
      return '<div class="token-limit-row" data-model="'+esc(model)+'">' +
        '<div class="token-limit-head"><span class="token-limit-model" title="'+esc(model)+'">'+esc(model)+'</span>' +
        '<button type="button" class="btn ghost sm" data-remove-token-limit="'+esc(model)+'"'+(disabled ? ' disabled' : '')+'>'+esc(t('移除'))+'</button></div>' +
        '<div class="token-limit-periods">' +
          periodField('total', '总 Token', item.total) +
          periodField('daily', '日 Token', item.daily) +
          periodField('weekly', '周 Token', item.weekly) +
          periodField('monthly', '月 Token', item.monthly) +
        '</div></div>';
    }).join('');
    target.querySelectorAll('.token-limit-period').forEach(syncTokenLimitPeriod);
  }

  function setKeyTokenLimitModels(models) {
    state.tokenLimitModels = [...new Set((models || []).map(value => String(value).trim()).filter(Boolean))].sort();
    if (!$('keyModalTokenLimitOptions').hidden) renderTokenLimitModelOptions();
  }

  function tokenLimitModelChoices() {
    const selected = new Set(collectModelTokenLimits().map(item => item.model));
    return (state.tokenLimitModels || []).filter(id => !selected.has(id));
  }

  function closeTokenLimitModelSearch() {
    const wrap = $('keyModalTokenLimitSearch');
    const panel = $('keyModalTokenLimitOptions');
    wrap.classList.remove('open');
    panel.hidden = true;
    panel.style.position = '';
    panel.style.left = '';
    panel.style.top = '';
    panel.style.bottom = '';
    panel.style.width = '';
    panel.style.minWidth = '';
    panel.style.maxWidth = '';
    panel.style.removeProperty('--token-limit-panel-width');
    panel.style.maxHeight = '';
    panel.style.overflow = '';
    if (panel.parentElement !== wrap) wrap.appendChild(panel);
  }

  function positionTokenLimitPanel() {
    const input = $('keyModalTokenLimitModel');
    const panel = $('keyModalTokenLimitOptions');
    const rect = input.getBoundingClientRect();
    const width = Math.round(rect.width);
    const spaceBelow = window.innerHeight - rect.bottom - 12;
    const spaceAbove = rect.top - 12;
    const openUp = spaceBelow < 132 && spaceAbove > spaceBelow;
    const available = openUp ? spaceAbove : spaceBelow;
    const maxHeight = Math.min(168, Math.max(96, available));
    panel.style.position = 'fixed';
    panel.style.left = Math.round(rect.left) + 'px';
    panel.style.setProperty('--token-limit-panel-width', width + 'px');
    panel.style.setProperty('width', width + 'px', 'important');
    panel.style.setProperty('min-width', width + 'px', 'important');
    panel.style.setProperty('max-width', width + 'px', 'important');
    panel.style.maxHeight = maxHeight + 'px';
    panel.style.overflow = 'auto';
    panel.style.zIndex = '80';
    if (openUp) {
      panel.style.top = 'auto';
      panel.style.bottom = (window.innerHeight - rect.top + 6) + 'px';
    } else {
      panel.style.bottom = 'auto';
      panel.style.top = (rect.bottom + 6) + 'px';
    }
  }

  function renderTokenLimitModelOptions() {
    const input = $('keyModalTokenLimitModel');
    const panel = $('keyModalTokenLimitOptions');
    const query = String(input.value || '').trim();
    const needle = query.toLocaleLowerCase();
    const matches = tokenLimitModelChoices().filter(id => !needle || id.toLocaleLowerCase().includes(needle));
    const rows = matches.map(id =>
      '<button class="custom-option" type="button" data-token-model="'+esc(id)+'">'+esc(id)+'</button>'
    );
    if (query && !matches.some(id => id === query)) {
      rows.unshift('<button class="custom-option" type="button" data-token-model="'+esc(query)+'"><span class="token-limit-use">'+esc(t('使用'))+'</span>'+esc(query)+'</button>');
    }
    panel.innerHTML = rows.length ? rows.join('') : '<div class="key-search-empty">'+esc(t('没有可添加的模型'))+'</div>';
    panel.querySelectorAll('[data-token-model]').forEach(button => button.addEventListener('click', event => {
      event.preventDefault();
      event.stopPropagation();
      addKeyTokenLimit(button.dataset.tokenModel);
    }));
  }

  function openTokenLimitModelSearch() {
    if ($('keyModalTokenLimitModel').disabled) return;
    closeCustomControls();
    const wrap = $('keyModalTokenLimitSearch');
    const panel = $('keyModalTokenLimitOptions');
    if (panel.parentElement !== document.body) document.body.appendChild(panel);
    wrap.classList.add('open');
    renderTokenLimitModelOptions();
    panel.hidden = false;
    positionTokenLimitPanel();
  }

  function addKeyTokenLimit(model) {
    model = String(model || '').trim();
    if (!model) {
      flash(t('请输入模型 ID 或 glob'), false);
      return;
    }
    const current = collectModelTokenLimits();
    if (current.some(item => item.model === model)) {
      flash(t('该模型已在 Token 限制列表中'), false);
      return;
    }
    current.push({
      model,
      total: { tokens: 0, mode: 'unlimited' },
      daily: { tokens: 0, mode: 'unlimited' },
      weekly: { tokens: 0, mode: 'unlimited' },
      monthly: { tokens: 0, mode: 'unlimited' },
    });
    renderKeyTokenLimits(current, $('keyModalMode').value === 'rotate');
    $('keyModalTokenLimitModel').value = '';
    closeTokenLimitModelSearch();
  }

  async function loadKeyModels(selectedModels) {
    setKeyModels(selectedModels);
    try {
      const available = await fetchAvailableModels();
      state.availableModels = modelIDs(available);
      setKeyModels(selectedModels);
    } catch (e) {
      $('keyModalModelsHint').textContent = '模型目录加载失败：' + e.message + '。可稍后重新打开弹窗。';
    }
  }

  function openKeyModal(mode, id) {
    const key = id ? state.keys.find(x => x.id === id) : null;
    const isManagedKey = mode === 'manage';
    const isRotation = mode === 'rotate';
    const titles = { create: '添加密钥', manage: '管理密钥', rotate: '轮换密钥' };
    const hints = {
      create: '设置额度和模型策略后创建凭据。',
      manage: '可编辑密钥策略，按需查看明文或轮换凭据。',
      rotate: '创建替代凭据并立即禁用旧密钥；历史使用记录会保留在旧密钥上。',
    };
    $('keyModalMode').value = mode;
    $('keyModalId').value = key ? key.id : '';
    $('keyModalTitle').textContent = titles[mode];
    $('keyModalHint').textContent = hints[mode];
    $('keyModalLabel').value = key ? (key.label || '') : '';
    $('keyModalQuotaUSD').value = key ? Number(key.quota_micro_usd || 0) / 1e6 : '';
    $('keyModalDailyQuotaUSD').value = key ? Number(key.daily_quota_micro_usd || 0) / 1e6 : '';
    $('keyModalWeeklyQuotaUSD').value = key ? Number(key.weekly_quota_micro_usd || 0) / 1e6 : '';
    $('keyModalMonthlyQuotaUSD').value = key ? Number(key.monthly_quota_micro_usd || 0) / 1e6 : '';
    $('keyModalMaxConcurrent').value = key ? (Number(key.max_concurrent_requests || 0) || '') : '';
    const allowedModels = key ? (key.allowed_models || []) : [];
    setKeyModels(allowedModels);
    $('keyModalEnabled').checked = !(key && !key.enabled);
    $('keyModalEnabled').disabled = isRotation;
    $('keyModalLabel').disabled = isRotation;
    $('keyModalQuotaUSD').disabled = isRotation;
    $('keyModalDailyQuotaUSD').disabled = isRotation;
    $('keyModalWeeklyQuotaUSD').disabled = isRotation;
    $('keyModalMonthlyQuotaUSD').disabled = isRotation;
    $('keyModalMaxConcurrent').disabled = isRotation;
    $('keyModalModels').disabled = isRotation;
    $('keyModalTokenLimitModel').disabled = isRotation;
    $('btnAddKeyTokenLimit').disabled = isRotation;
    setUnmatchedModelsMode(key ? key.unmatched_models_mode : 'available', isRotation);
    renderKeyTokenLimits(key ? (key.model_token_limits || []) : [], isRotation);
    const hasTokenLimits = Boolean(key && ((key.model_token_limits || []).length || key.unmatched_models_mode === 'disabled'));
    setTokenLimitsSectionVisible(hasTokenLimits, isRotation);
    $('keyModalEnabledWrap').classList.toggle('hidden', isRotation);
    $('keyModalCredentialSection').classList.toggle('hidden', mode === 'manage');
    $('keyMaterialWrap').classList.toggle('hidden', mode === 'manage');
    $('keyModalManageActions').classList.toggle('hidden', !isManagedKey);
    $('btnRotateManagedKey').classList.toggle('hidden', Boolean(key && key.revoked_at));
    $('keyMaterial').value = '';
    $('keyPlainResult').classList.add('hidden');
    $('btnSubmitKeyModal').textContent = mode === 'create' ? '创建密钥' : (mode === 'manage' ? '保存策略' : '确认轮换');
    refreshCustomControls();
    const modal = $('keyModal');
    modal.classList.add('open');
    modal.setAttribute('aria-hidden', 'false');
    $('keyModalLabel').focus();
    void loadKeyModels(allowedModels);
  }

  function closeKeyModal() {
    closeTokenLimitModelSearch();
    const modal = $('keyModal');
    modal.classList.remove('open');
    modal.setAttribute('aria-hidden', 'true');
  }

  function priceValue(price, camel, snake) {
    return price && (price[camel] != null ? price[camel] : price[snake]);
  }

  function openPriceModal(rule) {
    const price = rule && (rule.Price || rule.price) || {};
    const id = rule && (rule.ID || rule.id) || '';
    $('priceModalMode').value = rule ? 'edit' : 'create';
    $('priceModalTitle').textContent = rule ? '编辑价格规则' : '新增价格规则';
    $('priceId').value = id;
    $('priceId').disabled = Boolean(rule);
    $('priceKind').value = rule ? (rule.MatchKind || rule.match_kind) : 'exact';
    $('pricePattern').value = rule ? (rule.Pattern || rule.pattern) : '';
    $('pricePriority').value = rule ? (rule.Priority != null ? rule.Priority : rule.priority) : 100;
    $('priceIn').value = Number(priceValue(price, 'Input', 'input') || 0) / 1e6;
    $('priceOut').value = Number(priceValue(price, 'Output', 'output') || 0) / 1e6;
    $('priceCacheRead').value = Number(priceValue(price, 'CacheRead', 'cache_read') || 0) / 1e6;
    $('priceCacheCreation').value = Number(priceValue(price, 'CacheCreation', 'cache_creation') || 0) / 1e6;
    $('priceAccountingMode').value = priceValue(price, 'AccountingMode', 'accounting_mode') || '';
    $('priceBillingMode').value = (priceValue(price, 'BillingMode', 'billing_mode') === 'per_image') ? 'per_image' : 'token';
    $('pricePerImage').value = Number(priceValue(price, 'PerImage', 'per_image') || 0) / 1e6;
    $('priceEnabled').checked = ruleEnabled(rule);
    $('priceModelPicker').value = '';
    $('modelPriceStatus').textContent = rule ? '正在编辑「' + id + '」' : '价格来源：models.dev';
    fillPriceTiers(ruleTiers(rule));
    syncPriceBillingFields();
    refreshCustomControls();
    const modal = $('priceModal');
    modal.classList.add('open');
    modal.setAttribute('aria-hidden', 'false');
    $('priceId').focus();
  }

  function closePriceModal() {
    const modal = $('priceModal');
    modal.classList.remove('open');
    modal.setAttribute('aria-hidden', 'true');
  }

  async function submitKeyModal() {
    const mode = $('keyModalMode').value;
    const keyMaterial = mode === 'manage' ? '' : keyMaterialFromInput();
    const id = $('keyModalId').value;
    // empty or 0 = unlimited; always send a non-negative micro-USD amount
    const quota = mode === 'rotate' ? null : (microFromUSD($('keyModalQuotaUSD').value) ?? 0);
    const dailyQuota = mode === 'rotate' ? null : (microFromUSD($('keyModalDailyQuotaUSD').value) ?? 0);
    const weeklyQuota = mode === 'rotate' ? null : (microFromUSD($('keyModalWeeklyQuotaUSD').value) ?? 0);
    const monthlyQuota = mode === 'rotate' ? null : (microFromUSD($('keyModalMonthlyQuotaUSD').value) ?? 0);
    const maxConcurrentRaw = $('keyModalMaxConcurrent').value.trim();
    const maxConcurrent = mode === 'rotate' ? null : (maxConcurrentRaw === '' ? 0 : Number(maxConcurrentRaw));
    if (maxConcurrent !== null && (!Number.isInteger(maxConcurrent) || maxConcurrent < 0)) {
      throw new Error('最大并发请求数必须是非负整数');
    }
    let result;
    if (mode === 'create') {
      result = await api('POST', 'credit-manager/keys', {
        label: $('keyModalLabel').value.trim(),
        total_quota_micro_usd: quota,
        daily_quota_micro_usd: dailyQuota,
        weekly_quota_micro_usd: weeklyQuota,
        monthly_quota_micro_usd: monthlyQuota,
        max_concurrent_requests: maxConcurrent,
        enabled: $('keyModalEnabled').checked,
        allowed_models: selectedKeyModels(),
        ...collectKeyTokenLimitPayload(),
        key_material: keyMaterial,
      });
    } else if (mode === 'manage') {
      result = await api('POST', 'credit-manager/keys/update', {
        id,
        label: $('keyModalLabel').value,
        total_quota_micro_usd: quota,
        daily_quota_micro_usd: dailyQuota,
        weekly_quota_micro_usd: weeklyQuota,
        monthly_quota_micro_usd: monthlyQuota,
        max_concurrent_requests: maxConcurrent,
        enabled: $('keyModalEnabled').checked,
        set_allowed_models: true,
        allowed_models: selectedKeyModels(),
        set_model_token_limits: true,
        ...collectKeyTokenLimitPayload(),
      });
    } else {
      result = await api('POST', 'credit-manager/keys/rotate', { id, key_material: keyMaterial });
    }
    await reload();
    if (mode === 'manage') {
      closeKeyModal();
      flash('密钥策略已保存', true);
      return;
    }
    if (mode === 'create' && result.plaintext) {
      try {
        await copyText(result.plaintext);
        closeKeyModal();
        flash('密钥已创建并复制到剪贴板', true);
      } catch (_) {
        $('keyPlaintext').textContent = result.plaintext;
        $('keyPlainResult').classList.remove('hidden');
        flash('密钥已创建；自动复制失败，请手动复制', false);
      }
      return;
    }
    if (result.plaintext) {
      $('keyPlaintext').textContent = result.plaintext;
      $('keyPlainResult').classList.remove('hidden');
    }
    flash(mode === 'rotate' ? '密钥已轮换，旧密钥已禁用；请复制新密钥' : '密钥已创建；请复制明文', true);
  }

  function priceNumber(value) {
    const number = Number(value);
    return Number.isFinite(number) && number >= 0 ? number : 0;
  }

  function modelIDs(payload) {
    const items = Array.isArray(payload) ? payload : (payload && (payload.data || payload.models)) || [];
    return [...new Set(items.map(item => typeof item === 'string' ? item : item && (item.id || item.ID || item.name || item.alias)).filter(Boolean))].sort();
  }

  // Prefer first-party catalogs when the same model ID appears under many gateways.
  const modelsDevProviderRank = {
    openai: 100, anthropic: 100, google: 100, 'google-vertex': 95, xai: 100,
    deepseek: 100, mistral: 90, cohere: 90, meta: 90, 'amazon-bedrock': 85,
    azure: 85, together: 60, fireworks: 60, groq: 60, openrouter: 40,
    vercel: 35, github: 35, 'github-copilot': 35, opencode: 30,
  };

  function modelsDevBaseID(id) {
    const value = String(id || '').trim().toLowerCase().replace(/^~+/, '');
    const slash = value.lastIndexOf('/');
    return slash >= 0 ? value.slice(slash + 1) : value;
  }

  function modelsDevLooseID(id) {
    return modelsDevBaseID(id).replace(/[._]/g, '-');
  }

  function modelsDevCostKey(cost) {
    const c = cost || {};
    const base = [c.input, c.output, c.cache_read, c.cache_write].map(value => Number(value) || 0).join('|');
    return base + '|' + JSON.stringify(c.tiers || c.context_over_200k || null);
  }

  function formatContextLabel(tokens) {
    const n = Math.round(Number(tokens) || 0);
    if (n >= 1000 && n % 1000 === 0) return (n / 1000) + 'K';
    return String(n);
  }

  function modelsDevContextTiers(cost) {
    const c = cost || {};
    const out = [];
    const seen = new Set();
    const add = (threshold, rates) => {
      const n = Number(threshold);
      if (!(n > 0) || seen.has(n)) return;
      seen.add(n);
      out.push({
        kind: 'context',
        label: formatContextLabel(n),
        threshold: n,
        price: {
          input: priceNumber(rates && rates.input),
          output: priceNumber(rates && rates.output),
          cache_read: priceNumber(rates && (rates.cache_read != null ? rates.cache_read : rates.cacheRead)),
          cache_write: priceNumber(rates && (rates.cache_write != null ? rates.cache_write : rates.cache_creation)),
        },
      });
    };
    (Array.isArray(c.tiers) ? c.tiers : []).forEach(item => {
      const spec = (item && item.tier) || {};
      if (String(spec.type || '').toLowerCase() !== 'context') return;
      add(spec.size, item);
    });
    if (c.context_over_200k) add(200000, c.context_over_200k);
    return out.sort((a, b) => a.threshold - b.threshold);
  }

  function ruleTiers(rule) {
    return [...((rule && (rule.Tiers || rule.tiers)) || [])];
  }

  function tierKind(tier) {
    return String((tier && (tier.kind || tier.Kind)) || '').toLowerCase();
  }

  function tierService(tier) {
    return String((tier && (tier.service || tier.Service)) || '').toLowerCase();
  }

  function tierThreshold(tier) {
    return Number((tier && (tier.threshold != null ? tier.threshold : tier.Threshold)) || 0);
  }

  function tierPriceObj(tier) {
    return (tier && (tier.price || tier.Price)) || {};
  }

  function serviceTierMatches(tier, name) {
    return tierService(tier).split(',').map(part => part.trim()).includes(name);
  }

  function usdToMicroPrice(cost) {
    return {
      input: Math.round(priceNumber(cost && cost.input) * 1e6),
      output: Math.round(priceNumber(cost && cost.output) * 1e6),
      cache_read: Math.round(priceNumber(cost && cost.cache_read) * 1e6),
      cache_creation: Math.round(priceNumber(cost && (cost.cache_write != null ? cost.cache_write : cost.cache_creation)) * 1e6),
    };
  }

  function modelsDevTiersPayload(cost) {
    return modelsDevContextTiers(cost).map(tier => ({
      kind: 'context',
      label: tier.label,
      threshold: tier.threshold,
      price: usdToMicroPrice(tier.price),
    }));
  }

  function normalizeTierPayload(tier) {
    const price = tierPriceObj(tier);
    return {
      kind: tierKind(tier),
      label: String((tier && (tier.label || tier.Label)) || ''),
      threshold: tierThreshold(tier),
      service: tierService(tier),
      price: {
        input: Number(priceValue(price, 'Input', 'input') || 0),
        output: Number(priceValue(price, 'Output', 'output') || 0),
        cache_read: Number(priceValue(price, 'CacheRead', 'cache_read') || 0),
        cache_creation: Number(priceValue(price, 'CacheCreation', 'cache_creation') || 0),
      },
    };
  }

  function microToUsdField(value) {
    const n = Number(value || 0) / 1e6;
    return Number.isFinite(n) && n > 0 ? String(n) : '';
  }

  function emptyIfZero(value) {
    const n = Number(value);
    return Number.isFinite(n) && n > 0 ? String(n) : '';
  }

  function tierRateInputs(row) {
    return '<div class="price-rate-grid">' +
      '<label>input<input data-tier-in type="number" step="0.000001" min="0" value="'+esc(row.input || '')+'"/></label>' +
      '<label>output<input data-tier-out type="number" step="0.000001" min="0" value="'+esc(row.output || '')+'"/></label>' +
      '<label>缓存读<input data-tier-cache-read type="number" step="0.000001" min="0" value="'+esc(row.cache_read || '')+'"/></label>' +
      '<label>缓存写<input data-tier-cache-write type="number" step="0.000001" min="0" value="'+esc(row.cache_write || '')+'"/></label>' +
      '</div>';
  }

  function contextTierCardHTML(row) {
    return '<div class="price-tier-row">' +
      '<div class="price-tier-meta"><strong>上下文</strong><input data-tier-threshold type="number" min="1" step="1000" aria-label="阈值 Token" placeholder="272000" value="'+esc(row.threshold || '')+'"/><button class="btn ghost price-tier-remove" type="button" data-remove-context-tier aria-label="删除">×</button></div>' +
      tierRateInputs(row) +
      '</div>';
  }

  function renderContextTierCards(rows) {
    $('priceContextTiers').innerHTML = (rows || []).map(row => contextTierCardHTML(row)).join('');
  }

  function renderServiceTierCards(prices) {
    $('priceServiceTiers').innerHTML = ['fast', 'priority'].map(name => {
      const row = (prices && prices[name]) || {};
      return '<div class="price-tier-row" data-service="'+name+'"><div class="price-tier-meta"><strong>'+esc(name)+'</strong><span class="muted">service_tier</span></div>'+tierRateInputs(row)+'</div>';
    }).join('');
  }

  function collectTierPrice(card) {
    return {
      input: microFromUSD(card.querySelector('[data-tier-in]').value) || 0,
      output: microFromUSD(card.querySelector('[data-tier-out]').value) || 0,
      cache_read: microFromUSD(card.querySelector('[data-tier-cache-read]').value) || 0,
      cache_creation: microFromUSD(card.querySelector('[data-tier-cache-write]').value) || 0,
    };
  }

  function tierPriceFilled(price) {
    return ['input', 'output', 'cache_read', 'cache_creation'].some(key => Number(price[key] || 0) > 0);
  }

  function collectPriceTiers() {
    const tiers = [];
    document.querySelectorAll('#priceContextTiers .price-tier-row').forEach(card => {
      const threshold = Number(card.querySelector('[data-tier-threshold]').value || 0);
      const price = collectTierPrice(card);
      if (!(threshold > 0) || !tierPriceFilled(price)) return;
      tiers.push({ kind: 'context', label: formatContextLabel(threshold), threshold, price });
    });
    document.querySelectorAll('#priceServiceTiers .price-tier-row').forEach(card => {
      const service = String(card.dataset.service || '').trim();
      const price = collectTierPrice(card);
      if (!service || !tierPriceFilled(price)) return;
      tiers.push({ kind: 'service', label: service, service, price });
    });
    return tiers;
  }

  function fillPriceTiers(tiers, modelsDevCost) {
    const context = [];
    const servicePrices = { fast: {}, priority: {} };
    (tiers || []).forEach(tier => {
      if (tierKind(tier) === 'context') {
        const price = tierPriceObj(tier);
        context.push({
          threshold: tierThreshold(tier) || '',
          input: microToUsdField(priceValue(price, 'Input', 'input')),
          output: microToUsdField(priceValue(price, 'Output', 'output')),
          cache_read: microToUsdField(priceValue(price, 'CacheRead', 'cache_read')),
          cache_write: microToUsdField(priceValue(price, 'CacheCreation', 'cache_creation')),
        });
        return;
      }
      if (tierKind(tier) !== 'service') return;
      const usd = {
        input: microToUsdField(priceValue(tierPriceObj(tier), 'Input', 'input')),
        output: microToUsdField(priceValue(tierPriceObj(tier), 'Output', 'output')),
        cache_read: microToUsdField(priceValue(tierPriceObj(tier), 'CacheRead', 'cache_read')),
        cache_write: microToUsdField(priceValue(tierPriceObj(tier), 'CacheCreation', 'cache_creation')),
      };
      if (serviceTierMatches(tier, 'fast')) servicePrices.fast = usd;
      if (serviceTierMatches(tier, 'priority')) servicePrices.priority = usd;
    });
    if (!context.length && modelsDevCost) {
      modelsDevContextTiers(modelsDevCost).forEach(tier => {
        context.push({
          threshold: tier.threshold,
          input: emptyIfZero(tier.price.input),
          output: emptyIfZero(tier.price.output),
          cache_read: emptyIfZero(tier.price.cache_read),
          cache_write: emptyIfZero(tier.price.cache_write),
        });
      });
    }
    renderContextTierCards(context);
    renderServiceTierCards(servicePrices);
  }

  function tierDisplayName(tier) {
    if (tierKind(tier) === 'service') {
      const names = tierService(tier).split(',').map(part => part.trim()).filter(Boolean);
      return names.join(' / ') || String((tier && (tier.label || tier.Label)) || 'service');
    }
    return String((tier && (tier.label || tier.Label)) || '') || formatContextLabel(tierThreshold(tier));
  }

  function renderTierNotes(tiers, usd) {
    return (tiers || []).map(tier => {
      const price = usd ? (tier.price || {}) : tierPriceObj(tier);
      const inn = usd ? priceNumber(price.input) : formatMoney(priceValue(price, 'Input', 'input'));
      const out = usd ? priceNumber(price.output) : formatMoney(priceValue(price, 'Output', 'output'));
      const name = usd ? (tier.label || formatContextLabel(tier.threshold)) : tierDisplayName(tier);
      if (!name) return '';
      return '<div class="price-tier-note"><span>'+esc(name)+'</span><strong>'+esc(inn)+' / '+esc(out)+'</strong></div>';
    }).join('');
  }

  function modelsDevHasPrice(cost) {
    const c = cost || {};
    return ['input', 'output', 'cache_read', 'cache_write'].some(key => Number(c[key]) > 0);
  }

  function modelsDevModalities(model) {
    const raw = (model && model.modalities) || {};
    const list = value => [].concat(value || []).map(item => String(item || '').toLowerCase()).filter(Boolean);
    return { input: list(raw.input), output: list(raw.output) };
  }

  function isImageGenerationModel(modelID, modalities) {
    const id = String(modelID || '').toLowerCase();
    const output = (modalities && modalities.output) || [];
    if (/imagen|flux|dall-?e|stable-diffusion|gpt-image|chatgpt-image|grok-imagine|qwen-image|wan\d[\w.-]*image|image-preview|image-quality/.test(id)) return true;
    return output.includes('image') && !output.includes('text');
  }

  function modelsDevTokenPriced(cost, modelID, modalities) {
    if (!modelsDevHasPrice(cost)) return false;
    if (!isImageGenerationModel(modelID, modalities)) return true;
    const output = (modalities && modalities.output) || [];
    if (output.includes('text')) return true;
    const input = Number((cost || {}).input) || 0;
    const out = Number((cost || {}).output) || 0;
    return out > 0 || input >= 1;
  }

  function modelsDevMatchQuality(catalogID, providerID, query) {
    const id = String(catalogID || '').toLowerCase();
    const full = String(providerID + '/' + catalogID).toLowerCase();
    const base = modelsDevBaseID(id);
    const queryBase = modelsDevBaseID(query);
    if (id === query || full === query) return 100;
    if (base === query || base === queryBase) return 80;
    if (id.endsWith('/' + queryBase) || id.endsWith('.' + queryBase)) return 70;
    if (queryBase.length >= 4 && modelsDevLooseID(id) === modelsDevLooseID(query)) return 50;
    return 0;
  }

  function modelsDevCandidates(catalog, modelID) {
    const normalized = String(modelID).trim().toLowerCase();
    if (!normalized) return [];
    const matches = [];
    Object.entries(catalog || {}).forEach(([providerID, provider]) => {
      Object.values((provider && provider.models) || {}).forEach(model => {
        if (!model) return;
        const quality = modelsDevMatchQuality(model.id, providerID, normalized);
        if (!quality) return;
        const modalities = modelsDevModalities(model);
        const providerScore = modelsDevProviderRank[providerID] || 10;
        matches.push({
          providerID,
          model,
          quality,
          providerScore,
          priced: modelsDevHasPrice(model.cost),
          tokenPriced: modelsDevTokenPriced(model.cost, model.id || modelID, modalities),
        });
      });
    });
    return matches;
  }

  function resolveModelsDevMatch(candidates) {
    if (!candidates.length) return null;
    if (candidates.length === 1) return candidates[0];
    const qualities = [...new Set(candidates.map(item => item.quality))].sort((a, b) => b - a);
    let matched = null;
    for (const quality of qualities) {
      let pool = candidates.filter(item => item.quality === quality);
      const tokenPriced = pool.filter(item => item.tokenPriced);
      if (tokenPriced.length) pool = tokenPriced;
      else {
        const priced = pool.filter(item => item.priced);
        if (priced.length) pool = priced;
      }
      const bestProvider = Math.max(...pool.map(item => item.providerScore));
      const preferred = pool.filter(item => item.providerScore === bestProvider);
      const preferredCosts = new Set(preferred.map(item => modelsDevCostKey(item.model.cost)));
      if (preferred.length === 1 || preferredCosts.size === 1) {
        matched = preferred[0];
        break;
      }
      const poolCosts = new Set(pool.map(item => modelsDevCostKey(item.model.cost)));
      if (poolCosts.size === 1) {
        matched = preferred[0] || pool[0];
        break;
      }
    }
    if (!matched) return null;
    // Same price under a first-party catalog wins over gateway mirrors.
    const cost = modelsDevCostKey(matched.model.cost);
    return candidates
      .filter(item => (item.tokenPriced || item.priced) && modelsDevCostKey(item.model.cost) === cost)
      .sort((a, b) => Number(b.tokenPriced) - Number(a.tokenPriced) || b.providerScore - a.providerScore || b.quality - a.quality)[0] || matched;
  }

  function syncPriceBillingFields() {
    const perImage = $('priceBillingMode').value === 'per_image';
    $('priceTokenFields').classList.toggle('hidden', perImage);
    $('priceImageFields').classList.toggle('hidden', !perImage);
    $('priceTierSection').classList.toggle('hidden', perImage);
    $('priceAccountingWrap').classList.toggle('hidden', perImage);
  }

  function loadModelPrice(modelID) {
    const item = state.modelPrices && state.modelPrices[modelID];
    if (!item) return;
    const cost = item.cost || {};
    $('priceId').value = modelID;
    $('priceKind').value = 'exact';
    $('pricePattern').value = modelID;
    refreshCustomControls();
    if (item.tokenPriced) {
      $('priceBillingMode').value = 'token';
      $('priceIn').value = priceNumber(cost.input);
      $('priceOut').value = priceNumber(cost.output);
      $('priceCacheRead').value = priceNumber(cost.cache_read);
      $('priceCacheCreation').value = priceNumber(cost.cache_write);
      $('priceAccountingMode').value = /claude|anthropic/i.test(modelID) ? 'input_excludes_cache' : 'input_includes_cache';
      $('pricePerImage').value = 0;
      $('modelPriceStatus').textContent = item.imageGen
        ? '已回填 ' + modelID + ' 的 Token 价（图片输出按 Token，不是按张），来源：models.dev / ' + item.provider + '。'
        : '已回填 ' + modelID + ' 的价格，来源：models.dev / ' + item.provider + '。请核对后保存。';
      fillPriceTiers(collectPriceTiers().filter(tier => tier.kind === 'service'), item.cost);
    } else {
      $('priceBillingMode').value = 'per_image';
      $('priceIn').value = 0;
      $('priceOut').value = 0;
      $('priceCacheRead').value = 0;
      $('priceCacheCreation').value = 0;
      $('priceAccountingMode').value = '';
      $('pricePerImage').value = 0;
      $('modelPriceStatus').textContent = '「' + modelID + '」是出图模型，不能套用 Token 价。请填写每张 USD 后保存。';
    }
    syncPriceBillingFields();
  }

  function ruleEnabled(rule) {
    if (!rule) return true;
    return (rule.Enabled != null ? rule.Enabled : rule.enabled) !== false;
  }

  function pricingRuleIndex(rules) {
    return new Map((rules || []).filter(rule => (rule.MatchKind || rule.match_kind) === 'exact').map(rule => [rule.Pattern || rule.pattern, rule]));
  }

  function nextExactPriority(modelID, rules) {
    const winning = winningPricingRule(modelID, rules);
    const current = Number(winning && (winning.Priority != null ? winning.Priority : winning.priority) || 0);
    const winningID = winning ? String(winning.ID || winning.id || '') : '';
    if (winningID && winningID === modelID) return Math.max(100, current);
    return Math.max(100, current + 1);
  }

  function modelsDevPricingRule(modelID, item, existing) {
    const cost = item.cost || {};
    const contextTiers = modelsDevTiersPayload(cost);
    const serviceTiers = ruleTiers(existing).filter(tier => tierKind(tier) === 'service').map(normalizeTierPayload);
    const rule = {
      id: modelID,
      match_kind: 'exact',
      pattern: modelID,
      priority: nextExactPriority(modelID, (state.overview && state.overview.pricing) || []),
      enabled: true,
      price: {
        input: Math.round(priceNumber(cost.input) * 1e6),
        output: Math.round(priceNumber(cost.output) * 1e6),
        cache_read: Math.round(priceNumber(cost.cache_read) * 1e6),
        cache_creation: Math.round(priceNumber(cost.cache_write) * 1e6),
        accounting_mode: /claude|anthropic/i.test(modelID) ? 'input_excludes_cache' : 'input_includes_cache',
        billing_mode: 'token',
      },
      tiers: contextTiers.concat(serviceTiers),
    };
    if (existing) {
      rule.id = existing.ID || existing.id || modelID;
      rule.priority = Number(existing.Priority != null ? existing.Priority : existing.priority) || rule.priority;
      rule.enabled = ruleEnabled(existing);
      rule.price = cloneRulePrice(existing);
    }
    return rule;
  }

  async function syncModelsDevPrices(prices, rules) {
    const exactRules = pricingRuleIndex(rules);
    const ruleIDs = new Set((rules || []).map(rule => String(rule.ID || rule.id || '').trim()));
    const pending = Object.entries(prices).filter(([modelID, item]) => {
      if (!item || !item.tokenPriced) return false;
      const exact = exactRules.get(modelID);
      if (!exact && ruleIDs.has(modelID)) return false;
      if (!exact) return true;
      return !ruleTiers(exact).some(tier => tierKind(tier) === 'context') && modelsDevContextTiers(item.cost).length > 0;
    });
    let saved = 0;
    const failed = [];
    for (const [modelID, item] of pending) {
      try {
        await api('POST', 'credit-manager/pricing', modelsDevPricingRule(modelID, item, exactRules.get(modelID)));
        saved += 1;
      } catch (error) {
        failed.push(modelID + '（' + (error && error.message ? error.message : error) + '）');
      }
    }
    if (saved) {
      const result = await api('GET', 'credit-manager/pricing');
      state.overview = { ...(state.overview || {}), pricing: result.items || [] };
    }
    return { saved, skipped: Object.keys(prices).length - pending.length, failed };
  }

  async function loadModelCatalog() {
    const seq = state.tabLoadSeq;
    const status = $('modelCatalogStatus');
    const button = $('btnLoadModelCatalog');
    button.disabled = true;
    status.textContent = '正在读取当前代理模型与价格目录…';
    try {
      const catalogResult = await Promise.allSettled([
        fetchAvailableModels(),
        fetchModelsDevCatalog(false),
      ]);
      if (seq !== state.tabLoadSeq) return;
      if (catalogResult[0].status !== 'fulfilled') {
        throw catalogResult[0].reason || new Error('无法读取当前代理模型');
      }
      const available = catalogResult[0].value;
      const catalogError = catalogResult[1].status === 'fulfilled' ? '' : String((catalogResult[1].reason && catalogResult[1].reason.message) || catalogResult[1].reason || 'models.dev 价格目录不可用');
      const catalog = catalogResult[1].status === 'fulfilled' ? catalogResult[1].value.catalog : {};
      const models = modelIDs(available);
      const prices = {};
      const ambiguous = [];
      models.forEach(modelID => {
        const candidates = modelsDevCandidates(catalog, modelID);
        const matched = resolveModelsDevMatch(candidates);
        const modalities = matched ? modelsDevModalities(matched.model) : { input: [], output: [] };
        const imageGen = isImageGenerationModel(modelID, modalities);
        if (matched) {
          prices[modelID] = {
            provider: matched.providerID,
            cost: matched.model.cost || {},
            modalities,
            imageGen,
            tokenPriced: modelsDevTokenPriced(matched.model.cost, modelID, modalities),
          };
        } else if (candidates.length) {
          ambiguous.push(modelID);
        } else if (imageGen) {
          prices[modelID] = { provider: '', cost: {}, modalities, imageGen: true, tokenPriced: false };
        }
      });
      const sync = catalogError ? { saved: 0, skipped: Object.keys(prices).length, failed: [] } : await syncModelsDevPrices(prices, (state.overview && state.overview.pricing) || []);
      state.availableModels = models;
      state.modelPrices = prices;
      state.modelCatalogError = catalogError;
      const picker = $('priceModelPicker');
      picker.innerHTML = '<option value="">选择已匹配价格的当前模型</option>' + Object.keys(prices).sort().map(modelID =>
        '<option value="' + esc(modelID) + '">' + esc(modelID + ' · ' + prices[modelID].provider) + '</option>'
      ).join('');
      refreshCustomControl(picker);
      const tokenCount = Object.values(prices).filter(item => item.tokenPriced).length;
      const imageCount = Object.values(prices).filter(item => item.imageGen && !item.tokenPriced).length;
      $('modelCatalogCount').textContent = models.length + ' 个模型 · ' + tokenCount + ' 个 Token 价 · ' + imageCount + ' 个出图';
      const syncText = catalogError ? '价格目录暂不可用，未自动同步' : ('新增 ' + sync.saved + '，保留 ' + sync.skipped + (sync.failed.length ? '，失败 ' + sync.failed.length : ''));
      const ambiguousText = ambiguous.length ? '；' + ambiguous.length + ' 个未自动匹配' : '';
      status.textContent = catalogError
        ? ('已加载 ' + models.length + ' 个代理模型。' + syncText + '（' + catalogError + '）。仍可手动设置价格，稍后可重试同步。')
        : ('已加载 ' + models.length + ' 个模型，Token 价 ' + tokenCount + '，出图 ' + imageCount + '；' + syncText + ambiguousText + '。出图模型不会按 Token 价自动保存。');
      renderPricing((state.overview && state.overview.pricing) || []);
      if (sync.failed.length) throw new Error('部分 models.dev 价格未保存：' + sync.failed.join('；'));
    } finally {
      button.disabled = false;
    }
  }

  async function loadModelPrices() {
    await loadModelCatalog();
    $('modelPriceStatus').textContent = state.modelCatalogError
      ? ('价格目录暂不可用：' + state.modelCatalogError + '。仍可手动设置价格。')
      : '模型目录与价格规则已同步。选择模型可查看或调整 models.dev 价格。';
  }

  function ruleMatchesModel(rule, modelID) {
    const kind = rule.MatchKind || rule.match_kind;
    const pattern = String(rule.Pattern || rule.pattern || '');
    if (kind === 'exact') return pattern === modelID;
    if (kind === 'glob') {
      let escaped = '';
      for (const ch of pattern) {
        if (ch === '*') escaped += '[^/]*';
        else if (ch === '?') escaped += '[^/]';
        else escaped += ch.replace(/[.+^${}()|[\]\\]/g, '\\$&');
      }
      return new RegExp('^' + escaped + '$').test(modelID);
    }
    if (kind === 'regexp') {
      try { return new RegExp(pattern).test(modelID); } catch (_) { return false; }
    }
    return false;
  }

  function winningPricingRule(modelID, rules) {
    return [...(rules || [])].sort((a, b) => {
      const pa = Number(a.Priority != null ? a.Priority : a.priority) || 0;
      const pb = Number(b.Priority != null ? b.Priority : b.priority) || 0;
      if (pb !== pa) return pb - pa;
      return String(a.ID || a.id || '').localeCompare(String(b.ID || b.id || ''));
    }).find(rule => ruleMatchesModel(rule, modelID)) || null;
  }

  function modelIsDisabled(modelID, rules) {
    const winning = winningPricingRule(modelID, rules);
    return Boolean(winning) && !ruleEnabled(winning);
  }

  function cloneRulePrice(rule) {
    const price = (rule && (rule.Price || rule.price)) || {};
    return {
      input: Number(priceValue(price, 'Input', 'input') || 0),
      output: Number(priceValue(price, 'Output', 'output') || 0),
      reasoning: Number(priceValue(price, 'Reasoning', 'reasoning') || 0),
      cached: Number(priceValue(price, 'Cached', 'cached') || 0),
      cache_read: Number(priceValue(price, 'CacheRead', 'cache_read') || 0),
      cache_creation: Number(priceValue(price, 'CacheCreation', 'cache_creation') || 0),
      accounting_mode: priceValue(price, 'AccountingMode', 'accounting_mode') || '',
      billing_mode: priceValue(price, 'BillingMode', 'billing_mode') || 'token',
      per_image: Number(priceValue(price, 'PerImage', 'per_image') || 0),
    };
  }

  function modelPricingPayload(modelID, enabled) {
    const matched = state.modelPrices[modelID];
    if (matched && matched.tokenPriced) {
      return Object.assign(modelsDevPricingRule(modelID, matched), { enabled });
    }
    const rules = (state.overview && state.overview.pricing) || [];
    const inherited = winningPricingRule(modelID, rules);
    if (inherited && ruleEnabled(inherited)) {
      return {
        id: modelID,
        match_kind: 'exact',
        pattern: modelID,
        priority: nextExactPriority(modelID, rules),
        enabled,
        price: cloneRulePrice(inherited),
        tiers: ruleTiers(inherited).map(normalizeTierPayload),
      };
    }
    const imageGen = (matched && matched.imageGen) || isImageGenerationModel(modelID);
    return {
      id: modelID,
      match_kind: 'exact',
      pattern: modelID,
      priority: nextExactPriority(modelID, rules),
      enabled,
      price: {
        input: 0, output: 0, cache_read: 0, cache_creation: 0,
        accounting_mode: '',
        billing_mode: imageGen ? 'per_image' : 'token',
        per_image: 0,
      },
    };
  }

  function pricingModelImageGen(modelID) {
    const matched = state.modelPrices[modelID];
    return Boolean(matched && matched.imageGen) || isImageGenerationModel(modelID);
  }

  function pricingModelPriceAmount(modelID, rule, imageGen) {
    const price = rule && (rule.Price || rule.price) || {};
    const billing = priceValue(price, 'BillingMode', 'billing_mode');
    if (billing === 'per_image' || (imageGen && billing !== 'token')) {
      const perImage = Number(priceValue(price, 'PerImage', 'per_image') || 0);
      return perImage > 0 ? perImage : Number.POSITIVE_INFINITY;
    }
    const output = Number(priceValue(price, 'Output', 'output') || 0);
    const input = Number(priceValue(price, 'Input', 'input') || 0);
    if (output > 0 || input > 0) return output > 0 ? output : input;
    const cost = state.modelPrices[modelID] && state.modelPrices[modelID].cost;
    if (cost) {
      const out = priceNumber(cost.output);
      const inn = priceNumber(cost.input);
      if (out > 0 || inn > 0) return Math.round((out > 0 ? out : inn) * 1e6);
    }
    return Number.POSITIVE_INFINITY;
  }

  function pricingModelStatusRank(modelID, rule, rules) {
    if (modelIsDisabled(modelID, rules)) return 2;
    if (!rule) return 1;
    return 0;
  }

  function comparePricingModels(a, b, rulesByModel) {
    const rules = (state.overview && state.overview.pricing) || [];
    const ruleA = rulesByModel.get(a), ruleB = rulesByModel.get(b);
    const statusCmp = pricingModelStatusRank(a, ruleA, rules) - pricingModelStatusRank(b, ruleB, rules);
    if (statusCmp) return statusCmp;
    const imageA = pricingModelImageGen(a), imageB = pricingModelImageGen(b);
    if (imageA !== imageB) return imageA ? -1 : 1;
    const nameCmp = String(a).localeCompare(String(b), undefined, { numeric: true, sensitivity: 'base' });
    if (nameCmp) return nameCmp;
    return pricingModelPriceAmount(a, ruleA, imageA) - pricingModelPriceAmount(b, ruleB, imageB);
  }

  async function refreshPricingRules() {
    const result = await api('GET', 'credit-manager/pricing');
    const items = result.items || [];
    state.overview = { ...(state.overview || {}), pricing: items };
    renderPricing(items);
  }

  async function setModelPricingEnabled(modelID, enabled) {
    const rules = (state.overview && state.overview.pricing) || [];
    const exact = pricingRuleIndex(rules).get(modelID);
    const winning = winningPricingRule(modelID, rules);
    const exactID = exact ? String(exact.ID || exact.id || '') : '';
    const winningID = winning ? String(winning.ID || winning.id || '') : '';
    if (exactID && exactID === winningID) {
      await api('POST', 'credit-manager/pricing/enabled', { id: exactID, enabled });
    } else {
      await api('POST', 'credit-manager/pricing', modelPricingPayload(modelID, enabled));
    }
    await refreshPricingRules();
    flash(enabled ? '模型已启用' : '模型已禁用', true);
  }

  async function toggleModelPricingEnabled(modelID, enabled, control) {
    const wrap = control.closest('.key-switch');
    if (wrap) wrap.classList.add('is-busy');
    control.disabled = true;
    try {
      await setModelPricingEnabled(modelID, enabled);
      wrap && wrap.setAttribute('title', enabled ? t('启用') : t('禁用'));
    } catch (e) {
      control.checked = !enabled;
      flash(e.message, false);
    } finally {
      control.disabled = false;
      wrap && wrap.classList.remove('is-busy');
    }
  }

  function renderPricing(items) {
    const rules = items || [];
    const rulesByModel = pricingRuleIndex(rules);
    const models = [...new Set([...(state.availableModels || []), ...rulesByModel.keys()])].sort((a, b) => comparePricingModels(a, b, rulesByModel));
    if (!models.length) {
      $('pricingTable').innerHTML = '<p class="hint">尚未加载模型目录。点击“加载全部模型”后将同步当前代理模型和 models.dev 价格。</p>';
      return;
    }
    $('pricingTable').innerHTML = '<div class="table-scroll"><table class="pricing-table"><thead><tr><th>模型</th><th>models.dev 价格</th><th>当前规则</th><th>状态</th><th>操作</th></tr></thead><tbody>' +
      models.map(modelID => {
        const matched = state.modelPrices[modelID];
        const rule = rulesByModel.get(modelID);
        const winning = winningPricingRule(modelID, rules);
        const price = (rule && (rule.Price || rule.price)) || (winning && (winning.Price || winning.price)) || {};
        const imageGen = matched && matched.imageGen || isImageGenerationModel(modelID);
        const tokenPriced = matched && matched.tokenPriced;
        const enabled = !modelIsDisabled(modelID, rules);
        const pricePairs = tokenPriced ? [
          ['输入', matched.cost.input], ['输出', matched.cost.output], ['缓存读取', matched.cost.cache_read], ['缓存创建', matched.cost.cache_write],
        ].map(([label, value]) => '<span class="price-pair"><span>'+esc(label)+'</span><strong>'+esc(priceNumber(value))+'</strong></span>').join('') + renderTierNotes(modelsDevContextTiers(matched.cost), true) : (imageGen ? '<span class="muted">按张计费，不能套用 Token 价</span>' : '<span class="muted">未找到唯一匹配价格</span>');
        const ruleBilling = priceValue(price, 'BillingMode', 'billing_mode');
        const currentRule = rule ? '<div class="pricing-rule-id mono">'+esc(rule.ID || rule.id)+'</div><div class="hint">'+(ruleBilling === 'per_image' ? '每张 '+esc(formatMoney(priceValue(price, 'PerImage', 'per_image'))) : '输入 '+esc(formatMoney(priceValue(price, 'Input', 'input')))+' · 输出 '+esc(formatMoney(priceValue(price, 'Output', 'output'))))+'</div>'+renderTierNotes(ruleTiers(rule), false) : '<span class="muted">未设置</span>';
        const st = '<label class="key-switch" title="'+esc(enabled ? t('启用') : t('禁用'))+'"><input type="checkbox" role="switch" data-enable-price="'+esc(modelID)+'" aria-label="'+esc(t('启用'))+'"'+(enabled ? ' checked' : '')+'/><span class="key-switch-ui" aria-hidden="true"></span></label>';
        return '<tr'+(!enabled ? ' class="is-disabled"' : '')+'>' +
          '<td><div class="pricing-rule-id mono">'+esc(modelID)+'</div>'+(matched && matched.provider ? '<div class="hint">models.dev / '+esc(matched.provider)+(imageGen ? ' · 出图' : '')+'</div>' : (imageGen ? '<div class="hint">出图模型</div>' : ''))+'</td>' +
          '<td><div class="price-pairs">'+pricePairs+'</div></td>' +
          '<td>'+currentRule+'</td>' +
          '<td>'+st+'</td>' +
          '<td><div class="actions pricing-actions"><button class="btn soft sm" data-configure-model="'+esc(modelID)+'">'+(rule ? '编辑' : '设置价格')+'</button>'+(rule ? '<button class="btn danger sm" data-del-price="'+esc(rule.ID || rule.id)+'">删除</button>' : '')+'</div></td>' +
          '</tr>';
      }).join('') + '</tbody></table></div>';
    $('pricingTable').querySelectorAll('[data-enable-price]').forEach(input => input.addEventListener('change', () => toggleModelPricingEnabled(input.dataset.enablePrice, input.checked, input)));
    $('pricingTable').querySelectorAll('[data-configure-model]').forEach(btn => btn.addEventListener('click', () => {
      const modelID = btn.dataset.configureModel;
      const rule = rulesByModel.get(modelID);
      openPriceModal(rule);
      if (!rule) loadModelPrice(modelID);
    }));
    $('pricingTable').querySelectorAll('[data-del-price]').forEach(btn => btn.addEventListener('click', async () => {
      if (!confirm('删除价格规则 ' + btn.dataset.delPrice + ' ?')) return;
      try {
        await api('POST', 'credit-manager/pricing/delete', { id: btn.dataset.delPrice });
        flash('已删除价格规则', true);
        await reload();
      } catch (e) { flash(e.message, false); }
    }));
  }

  function normKeySum(x) {
    return {
      label: x.Label || x.label || '历史密钥',
      req: x.RequestCount || x.request_count || 0,
      cost: x.CostMicroUSD || x.cost_micro_usd || 0,
      inn: x.InputTokens || x.input_tokens || 0,
      out: x.OutputTokens || x.output_tokens || 0,
    };
  }
  function normModelSum(x) {
    return {
      model: x.Model || x.model,
      req: x.RequestCount || x.request_count || 0,
      cost: x.CostMicroUSD || x.cost_micro_usd || 0,
      inn: x.InputTokens || x.input_tokens || 0,
      out: x.OutputTokens || x.output_tokens || 0,
    };
  }

  function renderUsageModelFilter(items) {
    const select = $('usageModelFilter');
    const current = select.value;
    const models = [...new Set((items || []).map(item => {
      const value = item.Model != null ? item.Model : item.model;
      return String(value || '').trim();
    }).filter(Boolean))].sort((a, b) => a.localeCompare(b));
    select.innerHTML = '<option value="">全部已使用模型</option>' + models.map(model =>
      '<option value="'+esc(model)+'">'+esc(model)+'</option>'
    ).join('');
    if (models.includes(current)) select.value = current;
    refreshCustomControl(select);
  }

  function renderUsage(summary, recent) {
    renderUsageModelFilter((state.overview && state.overview.used_models) || []);
    const pageData = recent || {};
    const byKey = (summary.by_key || []).map(normKeySum);
    const byModel = (summary.by_model || []).map(normModelSum);
    const items = pageData.items || [];
    const totalRequests = byKey.reduce((sum, item) => sum + Number(item.req || 0), 0);
    const totalCost = byKey.reduce((sum, item) => sum + Number(item.cost || 0), 0);
    const totalInput = byKey.reduce((sum, item) => sum + Number(item.inn || 0), 0);
    const totalOutput = byKey.reduce((sum, item) => sum + Number(item.out || 0), 0);
    $('usageStats').innerHTML = [
      ['activity', '筛选请求数', totalRequests],
      ['coin', '筛选费用', formatMoney(totalCost)],
      ['trend', '输入 Token', formatTokens(totalInput)],
      ['layers', '输出 Token', formatTokens(totalOutput)],
    ].map(([icon, k, v]) => '<div class="stat"><div class="k"><span class="stat-icon" aria-hidden="true">'+uiIcon(icon)+'</span>'+esc(k)+'</div><div class="v">'+esc(v)+'</div></div>').join('');
    $('usageByKeyCount').textContent = byKey.length + ' ' + t('个密钥');
    $('usageByModelCount').textContent = byModel.length + ' ' + t('个模型');
    $('usageRecentCount').textContent = (Number(pageData.total || 0)).toLocaleString() + ' ' + t('条明细');
    const filterLabels = [];
    const filter = getUsageFilterValues();
    const rangeText = $('usageRangeFilter').selectedOptions[0].textContent;
    if (rangeText) filterLabels.push(rangeText);
    if (filter.plugin_key_id) filterLabels.push('密钥');
    if (filter.auth_id || filter.auth_index) filterLabels.push('账号');
    if (filter.model) filterLabels.push('模型: ' + filter.model);
    if (filter.source) filterLabels.push('来源: ' + filter.source);
    if (filter.min_cost_micro_usd || filter.max_cost_micro_usd) filterLabels.push('费用范围');
    if (filter.min_tokens || filter.max_tokens) filterLabels.push('Token 范围');
    $('usageFilterState').textContent = filterLabels.length ? filterLabels.join(' · ') : '未设置筛选';

    const emptyState = message => '<div class="empty-state"><span>'+esc(message)+'</span></div>';
    $('usageByKey').innerHTML = byKey.length ? '<div class="table-scroll"><table><thead><tr><th>密钥</th><th>请求数</th><th>费用 '+esc(currencyCode())+'</th><th>in/out tokens</th></tr></thead><tbody>' +
      byKey.map(x => '<tr><td><div><strong>'+esc(x.label||'(无标签)')+'</strong></div></td><td>'+esc(x.req)+'</td><td title="'+esc(moneyTitle(x.cost))+'">'+esc(formatMoney(x.cost))+'</td><td title="'+esc(tokenTitle(x.inn)+' / '+tokenTitle(x.out))+'">'+esc(formatTokens(x.inn))+' / '+esc(formatTokens(x.out))+'</td></tr>').join('') +
      '</tbody></table></div><p class="table-swipe-hint">左右滑动查看完整汇总</p>' : emptyState('当前筛选条件下暂无按密钥汇总');
    $('usageByModel').innerHTML = byModel.length ? '<div class="table-scroll"><table><thead><tr><th>模型</th><th>请求数</th><th>费用 '+esc(currencyCode())+'</th><th>in/out tokens</th></tr></thead><tbody>' +
      byModel.map(x => '<tr><td class="mono">'+esc(x.model)+'</td><td>'+esc(x.req)+'</td><td title="'+esc(moneyTitle(x.cost))+'">'+esc(formatMoney(x.cost))+'</td><td title="'+esc(tokenTitle(x.inn)+' / '+tokenTitle(x.out))+'">'+esc(formatTokens(x.inn))+' / '+esc(formatTokens(x.out))+'</td></tr>').join('') +
      '</tbody></table></div><p class="table-swipe-hint">左右滑动查看完整汇总</p>' : emptyState('当前筛选条件下暂无按模型汇总');
    const formatOptional = (value, formatter) => value == null || value === '' ? '—' : formatter(value);
    const formatMilliseconds = value => formatOptional(value, v => {
      const n = Number(v);
      return Number.isFinite(n) ? (n < 1000 ? n.toFixed(n < 100 ? 1 : 0) + ' ms' : (n / 1000).toFixed(2) + ' s') : '—';
    });
    const formatTPS = value => formatOptional(value, v => {
      const n = Number(v);
      return Number.isFinite(n) ? n.toFixed(2) : '—';
    });
    const cacheHit = u => Number(u.cache_read_tokens || 0) > 0 ? '是' : '否';
    const totalUsageTokens = u => usageTokens(u);
    const usageAuthCell = u => {
      const provider = String(u.auth_provider || u.auth_type || '').trim();
      const account = String(u.auth_label || u.auth_email || u.auth_name || u.auth_id || u.auth_index || '').trim();
      const providerDisplay = formatProviderDisplay(provider);
      const accountDisplay = formatAuthAccount(account, provider, providerDisplay);
      const display = formatAuthDisplay(provider, account) || t('未知账号');
      const raw = [provider, account].filter(Boolean).join(' · ') || display;
      const secondary = accountDisplay && providerDisplay && !sameAuthText(accountDisplay, providerDisplay) ? providerDisplay : '';
      const primary = accountDisplay || providerDisplay || display;
      return '<td class="usage-key" title="'+esc(raw)+'"><div class="usage-key-label"><strong>'+esc(primary)+'</strong>'+(secondary ? '<div class="mono">'+esc(secondary)+'</div>' : '')+'</div></td>';
    };

    $('usageRecent').innerHTML = items.length ? '<div class="table-scroll"><table class="usage-table"><thead><tr><th>时间</th><th>账号</th><th>模型名称</th><th>来源</th><th>执行器</th><th>结果</th><th>首字延迟</th><th>生成时间</th><th>TPS</th><th>思考强度</th><th>输入</th><th>输出</th><th>思考</th><th>缓存读取</th><th>缓存创建</th><th>总 Token 数</th><th>缓存命中</th><th>费用 '+esc(currencyCode())+'</th></tr></thead><tbody>' +
      items.map(u => {
        const settledCost = u.cost_micro_usd;
        const cell = (value) => '<td title="'+esc(tokenTitle(value))+'">'+esc(formatTokens(value))+'</td>';
        return '<tr><td class="mono" title="'+esc(u.created_at)+'">'+esc(formatDateTime(u.created_at))+'</td>'+usageAuthCell(u)+
          '<td class="mono model" title="'+esc(u.model)+'">'+esc(u.model)+'</td><td>'+esc(u.source || '—')+'</td><td class="mono">'+esc(u.executor_type || '—')+'</td><td>'+esc(u.result || '—')+'</td><td>'+esc(formatMilliseconds(u.first_token_latency_ms))+'</td><td>'+esc(formatMilliseconds(u.generation_duration_ms))+'</td><td>'+esc(formatTPS(u.tokens_per_second))+'</td><td>'+esc(u.thinking_intensity || '—')+'</td>'+
          cell(u.input_tokens || 0)+cell(u.output_tokens || 0)+cell(u.reasoning_tokens || 0)+cell(u.cache_read_tokens || 0)+cell(u.cache_creation_tokens || 0)+cell(totalUsageTokens(u))+
          '<td>'+esc(cacheHit(u))+'</td><td title="'+esc(moneyTitle(settledCost))+'">'+esc(formatMoney(settledCost))+'</td></tr>';
      }).join('') + '</tbody></table></div><p class="table-swipe-hint">左右滑动查看完整明细</p>' : emptyState('当前筛选条件下暂无使用明细');
    renderUsagePagination(pageData);
  }

  function renderUsagePagination(pageData) {
    const total = Number(pageData.total || 0);
    const page = Number(pageData.page || state.usagePage || 1);
    const pageSize = Number(pageData.page_size || state.usagePageSize || 50);
    const totalPages = Math.max(Number(pageData.total_pages || 0), 1);
    state.usagePage = page;
    state.usagePageSize = pageSize;
    const start = total ? (page - 1) * pageSize + 1 : 0;
    const end = Math.min(page * pageSize, total);
    $('usagePagination').innerHTML = '<span class="muted"><span>显示</span> '+start+'–'+end+'，<span>共</span> '+total.toLocaleString()+' <span>条</span></span>' +
      '<label>每页<select id="usagePageSize"><option value="25">25 '+t('条')+'</option><option value="50">50 '+t('条')+'</option><option value="100">100 '+t('条')+'</option><option value="200">200 '+t('条')+'</option></select></label>' +
      '<button class="btn ghost" id="btnUsagePrev" '+(page <= 1 ? 'disabled' : '')+'>'+uiIcon('chevronL')+esc(t('上一页'))+'</button>' +
      '<span class="muted"><span>第</span> '+page+' / '+totalPages+' <span>页</span></span>' +
      '<button class="btn ghost" id="btnUsageNext" '+(page >= totalPages ? 'disabled' : '')+'>'+esc(t('下一页'))+uiIcon('chevronR')+'</button>';
    $('usagePageSize').value = String(pageSize);
    initCustomControls($('usagePagination'));
    refreshCustomControl($('usagePageSize'));
    $('usagePageSize').addEventListener('change', async event => {
      try {
        state.usagePageSize = Number(event.target.value);
        state.usagePage = 1;
        await loadUsage();
      } catch (e) { flash(e.message, false); }
    });
    $('btnUsagePrev').addEventListener('click', async () => {
      try {
        state.usagePage = Math.max(1, page - 1);
        await loadUsage();
      } catch (e) { flash(e.message, false); }
    });
    $('btnUsageNext').addEventListener('click', async () => {
      try {
        state.usagePage = Math.min(totalPages, page + 1);
        await loadUsage();
      } catch (e) { flash(e.message, false); }
    });
  }

  function getUsageFilterValues() {
    const value = id => ($(id).value || '').trim();
    const toMicro = id => {
      const raw = value(id);
      if (!raw) return '';
      const number = Number(raw);
      if (!Number.isFinite(number) || number < 0) throw new Error('费用范围必须是非负数字');
      return String(Math.round(number * 1e6));
    };
    const range = $('usageRangeFilter').value === 'custom'
      ? { from: value('usageFromFilter'), to: value('usageToFilter') }
      : usageRangeDates();
    const auth = resolveAuthFilter(value('usageAuthFilter'));
    const filter = {
      plugin_key_id: resolveKeyFilter($('usageKeyFilter')),
      auth_id: auth.auth_id,
      auth_provider: auth.auth_provider,
      auth_index: auth.auth_index,
      model: value('usageModelFilter'),
      source: value('usageSourceFilter'),
      from: range.from,
      to: range.to,
      min_cost_micro_usd: toMicro('usageMinCost'),
      max_cost_micro_usd: toMicro('usageMaxCost'),
      min_tokens: value('usageMinTokens'),
      max_tokens: value('usageMaxTokens'),
    };
    if (filter.from) filter.from = new Date(filter.from).toISOString();
    if (filter.to) filter.to = new Date(filter.to).toISOString();
    if (filter.from && filter.to && filter.from > filter.to) throw new Error('开始时间不能晚于结束时间');
    return filter;
  }

  function usageQuery(includePagination) {
    const params = new URLSearchParams();
    Object.entries(getUsageFilterValues()).forEach(([key, value]) => {
      if (value !== '') params.set(key, value);
    });
    if (includePagination) {
      params.set('page', String(state.usagePage));
      params.set('page_size', String(state.usagePageSize));
    }
    return '?' + params.toString();
  }

  async function loadUsage(resetPage) {
    if (resetPage) state.usagePage = 1;
    const seq = state.tabLoadSeq;
    const [summary, recent] = await Promise.all([
      api('GET', 'credit-manager/usage/summary' + usageQuery(false)),
      api('GET', 'credit-manager/usage' + usageQuery(true)),
    ]);
    if (seq !== state.tabLoadSeq) return;
    state.usageSummary = summary;
    state.usageRecent = recent;
    renderUsage(summary, recent);
  }

  function authQuotaValue(item, key) {
    if (!item) return undefined;
    const camel = key.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
    const pascal = camel.charAt(0).toUpperCase() + camel.slice(1);
    for (const name of [key, camel, pascal]) if (item[name] != null) return item[name];
    return undefined;
  }

  function authQuotaTime(value) {
    if (!value) return '—';
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString();
  }

  function authQuotaText(value) {
    return value == null || value === '' ? '—' : String(value);
  }

  function authQuotaRatio(window) {
    const used = Number(authQuotaValue(window, 'used_ratio'));
    const remaining = Number(authQuotaValue(window, 'remaining_ratio'));
    const known = Number.isFinite(used) || Number.isFinite(remaining);
    const ratio = Number.isFinite(used) ? used : (Number.isFinite(remaining) ? 1 - remaining : 0);
    const percent = Math.round(Math.max(0, Math.min(100, ratio * 100)));
    const tone = percent >= 90 ? 'danger' : (percent >= 70 ? 'warn' : '');
    return { known, percent, tone };
  }

  function authQuotaTimeMs(value) {
    const parsed = Date.parse(value || '');
    return Number.isFinite(parsed) ? parsed : NaN;
  }

  function authQuotaIsWeekly(window) {
    const id = String(authQuotaValue(window, 'id') || '').toLowerCase();
    const label = String(authQuotaValue(window, 'label') || '').toLowerCase();
    const duration = Number(authQuotaValue(window, 'duration_seconds'));
    if (authQuotaIsExcluded(window)) return false;
    if (id.includes('weekly') || id.includes('seven_day') || id.includes('7d') || id.includes('secondary') || id === 'summary' || label.includes('week') || label.includes('secondary') || label.includes('summary') || label.includes('周') || label.includes('週')) return true;
    if (Number.isFinite(duration) && duration >= 500000 && duration <= 700000) return true;
    const start = authQuotaTimeMs(authQuotaValue(window, 'cycle_start_at'));
    const reset = authQuotaTimeMs(authQuotaValue(window, 'resets_at'));
    if (Number.isFinite(start) && Number.isFinite(reset)) {
      const hours = (reset - start) / 36e5;
      return hours >= 120 && hours <= 200;
    }
    return false;
  }

  function authQuotaIsExcluded(window) {
    const id = String(authQuotaValue(window, 'id') || '').toLowerCase();
    const label = String(authQuotaValue(window, 'label') || '').toLowerCase();
    const mode = String(authQuotaValue(window, 'mode') || '').toLowerCase();
    const unit = String(authQuotaValue(window, 'unit') || '').toLowerCase();
    return id.includes('on-demand') || id.includes('ondemand') || id.includes('monthly') || label.includes('on demand') || label.includes('按需') || label.includes('month') || label.includes('月额') || mode === 'balance' || mode === 'fixed' || unit === 'currency';
  }

  function authQuotaIsFiveHour(window) {
    if (authQuotaIsExcluded(window) || authQuotaIsWeekly(window)) return false;
    const id = String(authQuotaValue(window, 'id') || '').toLowerCase();
    const label = String(authQuotaValue(window, 'label') || '').toLowerCase();
    const duration = Number(authQuotaValue(window, 'duration_seconds'));
    if (id.includes('five_hour') || id.includes('5h') || id.includes('5-hour') || id.includes('primary') || label.includes('five hour') || label.includes('5h') || label.includes('5 小时') || label.includes('五小时')) return true;
    if (Number.isFinite(duration) && duration >= 10800 && duration <= 21600) return true;
    const start = authQuotaTimeMs(authQuotaValue(window, 'cycle_start_at'));
    const reset = authQuotaTimeMs(authQuotaValue(window, 'resets_at'));
    if (Number.isFinite(start) && Number.isFinite(reset)) {
      const hours = (reset - start) / 36e5;
      return hours >= 3 && hours <= 6;
    }
    return false;
  }

  function authQuotaPeriodLabel(window) {
    if (authQuotaIsFiveHour(window)) return '5 小时';
    if (authQuotaIsWeekly(window)) return '周额度';
    return '';
  }

  function authQuotaPeriodBadge(window, label) {
    const period = authQuotaPeriodLabel(window);
    if (!period) return '';
    const text = String(label || '');
    if (period === '周额度' && /周|週|week/i.test(text)) return '';
    if (period === '5 小时' && /小时|hour|5h|五小时|five/i.test(text)) return '';
    return period;
  }

  function authQuotaDisplayWindows(windows, selected) {
    const list = Array.isArray(windows) ? windows : [];
    const weekly = list.filter(authQuotaIsWeekly);
    const fiveHour = list.filter(authQuotaIsFiveHour);
    const selectedWeekly = weekly.filter(window => authQuotaWeekKey(window) === selected);
    if (fiveHour.length || weekly.length) return fiveHour.concat(selectedWeekly);
    const rest = list.filter(window => !authQuotaIsExcluded(window));
    return rest.length ? rest : list;
  }

  function authQuotaWeekKey(window) {
    const start = authQuotaTimeMs(authQuotaValue(window, 'cycle_start_at'));
    const reset = authQuotaTimeMs(authQuotaValue(window, 'resets_at'));
    if (Number.isFinite(start) && Number.isFinite(reset)) return start + ':' + reset;
    if (Number.isFinite(reset)) return 'reset:' + reset;
    if (Number.isFinite(start)) return 'start:' + start;
    return '';
  }

  function authQuotaShortTime(value) {
    const ms = authQuotaTimeMs(value);
    if (!Number.isFinite(ms)) return '—';
    const date = new Date(ms);
    const month = String(date.getMonth() + 1);
    const day = String(date.getDate());
    const hour = String(date.getHours()).padStart(2, '0');
    const minute = String(date.getMinutes()).padStart(2, '0');
    return month + '/' + day + ' ' + hour + ':' + minute;
  }

  function authQuotaWeekLabel(window, now) {
    const start = authQuotaTimeMs(authQuotaValue(window, 'cycle_start_at'));
    const reset = authQuotaTimeMs(authQuotaValue(window, 'resets_at'));
    const current = (Number.isFinite(start) && Number.isFinite(reset) && start <= now && now < reset) || (!Number.isFinite(start) && Number.isFinite(reset) && now < reset);
    return (current ? t('当前') + ' · ' : '') + authQuotaShortTime(authQuotaValue(window, 'cycle_start_at')) + ' → ' + authQuotaShortTime(authQuotaValue(window, 'resets_at'));
  }

  function collectAuthQuotaWeeks(windows, now) {
    const weeks = [];
    const seen = new Set();
    (Array.isArray(windows) ? windows : []).forEach(window => {
      if (!authQuotaIsWeekly(window)) return;
      const key = authQuotaWeekKey(window);
      if (!key || seen.has(key)) return;
      seen.add(key);
      const start = authQuotaTimeMs(authQuotaValue(window, 'cycle_start_at'));
      const reset = authQuotaTimeMs(authQuotaValue(window, 'resets_at'));
      weeks.push({ key, start, reset, label: authQuotaWeekLabel(window, now), current: (Number.isFinite(start) && Number.isFinite(reset) && start <= now && now < reset) || (!Number.isFinite(start) && Number.isFinite(reset) && now < reset) });
    });
    weeks.sort((a, b) => (Number.isFinite(b.reset) ? b.reset : Number.isFinite(b.start) ? b.start : 0) - (Number.isFinite(a.reset) ? a.reset : Number.isFinite(a.start) ? a.start : 0));
    return weeks;
  }

  function authQuotaItemKey(item) {
    return String(authQuotaValue(item, 'auth_id') || authQuotaValue(item, 'auth_index') || authQuotaValue(item, 'display_name') || '');
  }

  function selectedAuthQuotaWeek(itemKey, weeks) {
    if (!weeks.length) {
      delete state.authQuotaWeeks[itemKey];
      return '';
    }
    const current = weeks.find(week => week.current);
    if (!weeks.some(week => week.key === state.authQuotaWeeks[itemKey])) state.authQuotaWeeks[itemKey] = (current && current.key) || weeks[0].key;
    return state.authQuotaWeeks[itemKey];
  }

  function authQuotaLocalCost(window) {
    const usage = authQuotaValue(window, 'local_usage');
    const cost = Number(authQuotaValue(usage, 'estimated_cost_micro_usd'));
    return Number.isFinite(cost) && cost >= 0 ? cost : null;
  }

  function authQuotaLimitValue(window) {
    const limit = Number(authQuotaValue(window, 'limit'));
    if (Number.isFinite(limit) && limit > 0) return limit;
    return String(authQuotaValue(window, 'unit') || '') === 'percentage' ? 100 : NaN;
  }

  function authQuotaRemainingRatio(window) {
    const ratio = Number(authQuotaValue(window, 'remaining_ratio'));
    if (Number.isFinite(ratio) && ratio >= 0) return ratio;
    const remaining = Number(authQuotaValue(window, 'remaining'));
    const limit = authQuotaLimitValue(window);
    if (Number.isFinite(remaining) && remaining >= 0 && Number.isFinite(limit) && limit > 0) return remaining / limit;
    return NaN;
  }

  function authQuotaObservedRatio(window) {
    const observed = Number(authQuotaValue(window, 'observed_used'));
    if (!Number.isFinite(observed) || observed <= 0) return NaN;
    const limit = authQuotaLimitValue(window);
    if (Number.isFinite(limit) && limit > 0) return observed / limit;
    const used = Number(authQuotaValue(window, 'used'));
    const usedRatio = Number(authQuotaValue(window, 'used_ratio'));
    if (Number.isFinite(used) && used > 0 && Number.isFinite(usedRatio) && usedRatio > 0) return observed * usedRatio / used;
    return NaN;
  }

  function authQuotaCostForecast(windows) {
    const minObservedRatio = 0.005;
    let usedMicro = 0;
    let remainingMicro = 0;
    let remainingKnown = false;
    (Array.isArray(windows) ? windows : []).forEach(window => {
      const cost = authQuotaLocalCost(window);
      if (cost == null) return;
      usedMicro += cost;
      if (!(cost > 0)) return;
      const observedRatio = authQuotaObservedRatio(window);
      const remainingRatio = authQuotaRemainingRatio(window);
      if (!Number.isFinite(observedRatio) || observedRatio < minObservedRatio || !Number.isFinite(remainingRatio) || remainingRatio < 0) return;
      remainingMicro += cost * remainingRatio / observedRatio;
      remainingKnown = true;
    });
    return {
      used: formatMoney(usedMicro),
      remaining: remainingKnown ? formatMoney(remainingMicro) : '—',
      available: remainingKnown ? formatMoney(usedMicro + remainingMicro) : '—'
    };
  }

  function authQuotaAmount(window, key) {
    if (key === 'used' || key === 'remaining') {
      const ratioKey = key === 'used' ? 'used_ratio' : 'remaining_ratio';
      const ratio = Number(authQuotaValue(window, ratioKey));
      if (Number.isFinite(ratio)) return Math.round(Math.max(0, Math.min(100, ratio * 100))) + '%';
      const limit = authQuotaLimitValue(window);
      const raw = Number(authQuotaValue(window, key));
      if (Number.isFinite(raw) && Number.isFinite(limit) && limit > 0) return Math.round(Math.max(0, Math.min(100, raw / limit * 100))) + '%';
      if (String(authQuotaValue(window, 'unit') || '') === 'percentage' && Number.isFinite(raw)) return Math.round(Math.max(0, Math.min(100, raw))) + '%';
    }
    const value = authQuotaValue(window, key);
    if (value == null || value === '') return '—';
    const unit = String(authQuotaValue(window, 'unit') || '');
    const number = Number(value);
    if (!Number.isFinite(number)) return authQuotaText(value);
    if (unit === 'percentage') return Math.round(number) + '%';
    if (unit === 'currency') return formatMoney(Math.round(number * 1e6));
    return authQuotaText(value);
  }



  function authQuotaPlanName(value) {
    const raw = String(value || '').trim();
    if (!raw) return '';
    const key = raw.toLowerCase().replace(/[_\s]+/g, '-');
    const names = {
      plus:'Plus', pro:'Pro', team:'Team', free:'Free', business:'Business', enterprise:'Enterprise',
      max:'Max', 'max-5x':'Max 5x', 'max-20x':'Max 20x', go:'Go', standard:'Standard', legacy:'Legacy', 'super-grok':'SuperGrok'
    };
    if (names[key]) return names[key];
    if (key.includes('max-20') || key.includes('max20')) return 'Max 20x';
    if (key.includes('max-5') || key.includes('max5')) return 'Max 5x';
    return names[raw.toLowerCase()] || raw;
  }

  function authQuotaProviderIsBrand(provider) {
    const key = String(provider || '').trim().toLowerCase();
    return key === 'xai' || key === 'grok' || key === 'codex' || key === 'openai'
      || key === 'claude' || key === 'anthropic' || key === 'antigravity' || key === 'google'
      || key === 'gemini' || key === 'kimi' || key === 'moonshot';
  }

  function authQuotaProviderName(provider) {
    return formatProviderDisplay(provider) || t('未知提供商');
  }

  function syncAuthQuotaProviderFilter(providers) {
    const select = $('authQuotaProviderFilter');
    if (!select) return;
    const list = Array.from(new Set((Array.isArray(providers) ? providers : []).map(provider => String(provider || '').trim()).filter(Boolean))).sort((a, b) => authQuotaProviderName(a).localeCompare(authQuotaProviderName(b), 'zh'));
    if (state.authQuotaProvider && !list.includes(state.authQuotaProvider)) state.authQuotaProvider = '';
    select.innerHTML = '<option value="">全部平台</option>' + list.map(provider => '<option value="'+esc(provider)+'">'+esc(authQuotaProviderName(provider))+'</option>').join('');
    select.value = state.authQuotaProvider;
  }

  function authQuotaQuery() {
    const params = new URLSearchParams();
    params.set('page', String(state.authQuotaPage || 1));
    params.set('page_size', String(state.authQuotaPageSize || 12));
    if (state.authQuotaProvider) params.set('provider', state.authQuotaProvider);
    const q = String(state.authQuotaName || '').trim();
    if (q) params.set('q', q);
    return params.toString();
  }

  function authQuotaIcon(name) {
    return uiIcon(name);
  }

  function authQuotaProviderIcon(provider) {
    const icons = {
      codex: '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M19.503 0H4.496A4.496 4.496 0 000 4.496v15.007A4.496 4.496 0 004.496 24h15.007A4.496 4.496 0 0024 19.503V4.496A4.496 4.496 0 0019.503 0z" fill="#fff"/><path d="M9.064 3.344a4.578 4.578 0 012.285-.312c1 .115 1.891.54 2.673 1.275.01.01.024.017.037.021a.09.09 0 00.043 0 4.55 4.55 0 013.046.275l.047.022.116.057a4.581 4.581 0 012.188 2.399c.209.51.313 1.041.315 1.595a4.24 4.24 0 01-.134 1.223.123.123 0 00.03.115c.594.607.988 1.33 1.183 2.17.289 1.425-.007 2.71-.887 3.854l-.136.166a4.548 4.548 0 01-2.201 1.388.123.123 0 00-.081.076c-.191.551-.383 1.023-.74 1.494-.9 1.187-2.222 1.846-3.711 1.838-1.187-.006-2.239-.44-3.157-1.302a.107.107 0 00-.105-.024c-.388.125-.78.143-1.204.138a4.441 4.441 0 01-1.945-.466 4.544 4.544 0 01-1.61-1.335c-.152-.202-.303-.392-.414-.617a5.81 5.81 0 01-.37-.961 4.582 4.582 0 01-.014-2.298.124.124 0 00.006-.056.085.085 0 00-.027-.048 4.467 4.467 0 01-1.034-1.651 3.896 3.896 0 01-.251-1.192 5.189 5.189 0 01.141-1.6c.337-1.112.982-1.985 1.933-2.618.212-.141.413-.251.601-.33.215-.089.43-.164.646-.227a.098.098 0 00.065-.066 4.51 4.51 0 01.829-1.615 4.535 4.535 0 011.837-1.388zm3.482 10.565a.637.637 0 000 1.272h3.636a.637.637 0 100-1.272h-3.636zM8.462 9.23a.637.637 0 00-1.106.631l1.272 2.224-1.266 2.136a.636.636 0 101.095.649l1.454-2.455a.636.636 0 00.005-.64L8.462 9.23z" fill="url(#auth-codex-fill)"/><defs><linearGradient id="auth-codex-fill" x1="12" x2="12" y1="3" y2="21" gradientUnits="userSpaceOnUse"><stop stop-color="#B1A7FF"/><stop offset=".5" stop-color="#7A9DFF"/><stop offset="1" stop-color="#3941FF"/></linearGradient></defs></svg>',
      claude: '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path fill="#D97757" fill-rule="nonzero" d="M4.709 15.955l4.72-2.647.08-.23-.08-.128H9.2l-.79-.048-2.698-.073-2.339-.097-2.266-.122-.571-.121L0 11.784l.055-.352.48-.321.686.06 1.52.103 2.278.158 1.652.097 2.449.255h.389l.055-.157-.134-.098-.103-.097-2.358-1.596-2.552-1.688-1.336-.972-.724-.491-.364-.462-.158-1.008.656-.722.881.06.225.061.893.686 1.908 1.476 2.491 1.833.365.304.145-.103.019-.073-.164-.274-1.355-2.446-1.446-2.49-.644-1.032-.17-.619a2.97 2.97 0 01-.104-.729L6.283.134 6.696 0l.996.134.42.364.62 1.414 1.002 2.229 1.555 3.03.456.898.243.832.091.255h.158V9.01l.128-1.706.237-2.095.23-2.695.08-.76.376-.91.747-.492.584.28.48.685-.067.444-.286 1.851-.559 2.903-.364 1.942h.212l.243-.242.985-1.306 1.652-2.064.73-.82.85-.904.547-.431h1.033l.76 1.129-.34 1.166-1.064 1.347-.881 1.142-1.264 1.7-.79 1.36.073.11.188-.02 2.856-.606 1.543-.28 1.841-.315.833.388.091.395-.328.807-1.969.486-2.309.462-3.439.813-.042.03.049.061 1.549.146.662.036h1.622l3.02.225.79.522.474.638-.079.485-1.215.62-1.64-.389-3.829-.91-1.312-.329h-.182v.11l1.093 1.068 2.006 1.81 2.509 2.33.127.578-.322.455-.34-.049-2.205-1.657-.851-.747-1.926-1.62h-.128v.17l.444.649 2.345 3.521.122 1.08-.17.353-.608.213-.668-.122-1.374-1.925-1.415-2.167-1.143-1.943-.14.08-.674 7.254-.316.37-.729.28-.607-.461-.322-.747.322-1.476.389-1.924.315-1.53.286-1.9.17-.632-.012-.042-.14.018-1.434 1.967-2.18 2.945-1.726 1.845-.414.164-.717-.37.067-.662.401-.589 2.388-3.036 1.44-1.882.93-1.086-.006-.158h-.055L4.132 18.56l-1.13.146-.487-.456.061-.746.231-.243 1.908-1.312-.006.006z"/></svg>',
      antigravity: '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" fill="currentColor" fill-rule="evenodd"><path d="M21.751 22.607c1.34 1.005 3.35.335 1.508-1.508C17.73 15.74 18.904 1 12.037 1 5.17 1 6.342 15.74.815 21.1c-2.01 2.009.167 2.511 1.507 1.506 5.192-3.517 4.857-9.714 9.715-9.714 4.857 0 4.522 6.197 9.714 9.715z"/></svg>',
      kimi: '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><rect width="24" height="24" rx="6" fill="#fff"/><path d="M21.846 0a1.923 1.923 0 110 3.846H20.15a.226.226 0 01-.227-.226V1.923C19.923.861 20.784 0 21.846 0z" fill="#1783FF"/><path d="M11.065 11.199l7.257-7.2c.137-.136.06-.41-.116-.41H14.3a.164.164 0 00-.117.051l-7.82 7.756c-.122.12-.302.013-.302-.179V3.82c0-.127-.083-.23-.185-.23H3.186c-.103 0-.186.103-.186.23V19.77c0 .128.083.23.186.23h2.69c.103 0 .186-.102.186-.23v-3.25c0-.069.025-.135.069-.178l2.424-2.406a.158.158 0 01.205-.023l6.484 4.772a7.677 7.677 0 003.453 1.283c.108.012.2-.095.2-.23v-3.06c0-.117-.07-.212-.164-.227a5.028 5.028 0 01-2.027-.807l-5.613-4.064c-.117-.078-.132-.279-.028-.381z" fill="#000"/></svg>',
      xai: '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" fill="currentColor" fill-rule="evenodd"><path d="M6.469 8.776L16.512 23h-4.464L2.005 8.776H6.47zm-.004 7.9l2.233 3.164L6.467 23H2l4.465-6.324zM22 2.582V23h-3.659V7.764L22 2.582zM22 1l-9.952 14.095-2.233-3.163L17.533 1H22z"/></svg>',
      default: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><circle cx="6" cy="8" r="2.7"/><path d="M8.7 8h5M11.4 8v2.4"/></svg>'
    };
    const key = String(provider || '').toLowerCase();
    const map = { openai: 'codex', chatgpt: 'codex', anthropic: 'claude', google: 'antigravity', gemini: 'antigravity', moonshot: 'kimi', grok: 'xai' };
    return icons[map[key] || key] || icons.default;
  }

  function authQuotaBadge(status) {
    switch (String(status || '').toLowerCase()) {
      case 'fresh': return { tone: 'ok', text: '最新' };
      case 'stale': return { tone: 'warn', text: '缓存过期' };
      case 'idle': return { tone: 'warn', text: '未同步' };
      case 'unavailable': return { tone: 'bad', text: '不可用' };
      default: return { tone: 'warn', text: authQuotaText(status) };
    }
  }

  function renderAuthQuotas(result) {
    if (result) state.authQuotas = result;
    const payload = state.authQuotas || {};
    const items = Array.isArray(authQuotaValue(payload, 'items')) ? authQuotaValue(payload, 'items') : [];
    const providers = authQuotaValue(payload, 'providers');
    syncAuthQuotaProviderFilter(providers);
    renderAuthQuotaPagination(payload);
    if (!items.length) {
      const total = Number(authQuotaValue(payload, 'total') || 0);
      $('authQuotaList').innerHTML = emptyState(total || state.authQuotaProvider || String(state.authQuotaName || '').trim() ? '没有符合筛选条件的认证额度' : '当前没有可用的认证额度数据');
      return;
    }
    const now = Date.now();
    $('authQuotaList').innerHTML = items.map(item => {
      const itemKey = authQuotaItemKey(item);
      const windows = authQuotaValue(item, 'windows');
      const weeks = collectAuthQuotaWeeks(windows, now);
      const selected = selectedAuthQuotaWeek(itemKey, weeks);
      const status = authQuotaValue(item, 'status');
      const badge = authQuotaBadge(status);
      const error = authQuotaValue(item, 'error') ?? authQuotaValue(item, 'last_error');
      const visible = authQuotaDisplayWindows(windows, selected);
      const cards = visible.length ? visible.map(window => {
        const ratio = authQuotaRatio(window);
        const label = authQuotaText(authQuotaValue(window, 'label'));
        const period = authQuotaPeriodBadge(window, label);
        const progress = ratio.known ? t('已用')+' '+ratio.percent+'%' : '—';
        const progressClass = ratio.known ? ratio.tone : 'unknown';
        return '<section class="auth-quota-window-card">'+
          '<div class="auth-quota-window-head"><div class="auth-quota-window-name" title="'+esc(label)+'">'+esc(label)+(period ? '<span class="auth-quota-period">'+esc(period)+'</span>' : '')+'</div><span class="auth-quota-window-reset" title="'+esc(t('重置时间')+' '+authQuotaTime(authQuotaValue(window, 'resets_at')))+'">'+esc(authQuotaShortTime(authQuotaValue(window, 'resets_at')))+'</span><span class="auth-quota-window-pct">'+esc(progress)+'</span></div>'+
          '<div class="auth-quota-bar '+progressClass+'" style="--quota-progress:'+ratio.percent+'%" role="progressbar" aria-valuemin="0" aria-valuemax="100"'+(ratio.known ? ' aria-valuenow="'+ratio.percent+'"' : '')+' aria-label="'+esc(label)+' '+progress+'"></div>'+
          '</section>';
      }).join('') : '<div class="empty-state">该额度周暂无窗口</div>';
      const weeklyForCost = (Array.isArray(windows) ? windows : []).filter(window => authQuotaIsWeekly(window) && authQuotaWeekKey(window) === selected);
      const costs = authQuotaCostForecast(weeklyForCost.length ? weeklyForCost : visible);
      const weekSelect = weeks.length
        ? '<label class="auth-quota-filter auth-quota-week"><select class="auth-quota-week-select" data-auth-id="'+esc(itemKey)+'" title="额度周">'+weeks.map(week => '<option value="'+esc(week.key)+'"'+(week.key === selected ? ' selected' : '')+'>'+esc(week.label)+'</option>').join('')+'</select></label>'
        : '<label class="auth-quota-filter auth-quota-week is-disabled"><select disabled title="额度周"><option>暂无额度周</option></select></label>';
      const refreshing = !!state.authQuotaRefreshing[itemKey];
      const maxConcurrent = Math.max(0, Number(authQuotaValue(item, 'max_concurrent_requests') || 0) || 0);
      const activeRequests = Math.max(0, Number(authQuotaValue(item, 'active_requests') || 0) || 0);
      const concurrentValue = maxConcurrent > 0 ? String(maxConcurrent) : '';
      const concurrentInput = '<div class="auth-quota-cost auth-quota-concurrency"><span class="auth-quota-concurrency-label"><span class="auth-quota-concurrency-icon" aria-hidden="true">'+authQuotaIcon('bolt')+'</span>'+esc(t('并发')+' · '+t('在途'))+'<b class="auth-quota-active'+(activeRequests > 0 ? ' is-active' : '')+'" title="在途请求数">'+String(activeRequests)+'</b></span><input class="auth-quota-concurrency-input" type="number" min="0" step="1" inputmode="numeric" placeholder="不限制" value="'+esc(concurrentValue)+'" data-provider="'+esc(authQuotaValue(item, 'provider') || '')+'" data-auth-id="'+esc(authQuotaValue(item, 'auth_id') || '')+'" data-auth-index="'+esc(authQuotaValue(item, 'auth_index') || '')+'" data-item-key="'+esc(itemKey)+'" title="最大并发请求数，0 或不填为不限制"></div>';
      const reloadBtn = '<button type="button" class="btn sm ghost auth-quota-reload" data-provider="'+esc(authQuotaValue(item, 'provider') || '')+'" data-auth-id="'+esc(authQuotaValue(item, 'auth_id') || '')+'" data-auth-index="'+esc(authQuotaValue(item, 'auth_index') || '')+'" data-item-key="'+esc(itemKey)+'"'+(refreshing ? ' disabled' : '')+' title="重新加载"><span class="auth-quota-reload-icon" aria-hidden="true">'+authQuotaIcon('refresh')+'</span><span>'+(refreshing ? '加载中' : '重新加载')+'</span></button>';
      const plan = authQuotaPlanName(authQuotaValue(item, 'plan'));
      const planLabel = plan ? '<span class="auth-quota-plan" title="订阅类型">'+esc(plan)+'</span>' : '';
      const provider = String(authQuotaValue(item, 'provider') || '');
      return '<article class="card auth-quota-card'+(refreshing ? ' is-refreshing' : '')+'" data-provider="'+esc(provider || 'unknown')+'"><header class="auth-quota-header"><div class="auth-quota-identity"><div class="auth-quota-identity-row"><div class="auth-quota-identity-main"><span class="auth-quota-provider-icon" aria-hidden="true">'+authQuotaProviderIcon(provider)+'</span><p class="auth-quota-provider'+(authQuotaProviderIsBrand(provider) ? '' : ' is-custom')+'" title="'+esc(provider || t('未知提供商'))+'">'+esc(authQuotaProviderName(provider))+'</p>'+planLabel+'</div><span class="badge '+badge.tone+'">'+esc(badge.text)+'</span></div><div class="auth-quota-account-row"><h2 class="auth-quota-title" title="'+esc(authQuotaValue(item, 'display_name') || t('未命名认证'))+'">'+esc(authQuotaValue(item, 'display_name') || t('未命名认证'))+'</h2><p class="auth-quota-sync" title="上次同步 '+esc(authQuotaTime(authQuotaValue(item, 'last_success_at')))+'"><span class="auth-quota-sync-icon" aria-hidden="true">'+authQuotaIcon('clock')+'</span><span>同步</span> '+esc(authQuotaShortTime(authQuotaValue(item, 'last_success_at')))+'</p></div></div><div class="auth-quota-cost-grid"><div class="auth-quota-cost"><span>'+authQuotaIcon('coin')+esc(t('当前费用'))+'</span><strong title="'+esc(costs.used)+'">'+esc(costs.used)+'</strong></div><div class="auth-quota-cost"><span>'+authQuotaIcon('wallet')+esc(t('预估剩余'))+'</span><strong title="'+esc(costs.remaining)+'">'+esc(costs.remaining)+'</strong></div><div class="auth-quota-cost"><span>'+authQuotaIcon('trend')+esc(t('预计可用'))+'</span><strong title="'+esc(costs.available)+'">'+esc(costs.available)+'</strong></div>'+concurrentInput+'</div><div class="auth-quota-header-tools">'+weekSelect+reloadBtn+'</div></header>'+(error ? '<div class="auth-quota-error">'+esc(error)+'</div>' : '')+'<div class="auth-quota-window-grid">'+cards+'</div></article>';
    }).join('');
  }

  async function loadAuthQuotas() {
    const seq = ++state.tabLoadSeq;
    const result = await api('GET', 'credit-manager/auth-quotas?' + authQuotaQuery());
    if (seq !== state.tabLoadSeq) return;
    renderAuthQuotas(result);
  }
  function replaceAuthQuotaItem(item) {
    const current = state.authQuotas && typeof state.authQuotas === 'object' ? state.authQuotas : {};
    const items = Array.isArray(authQuotaValue(current, 'items')) ? authQuotaValue(current, 'items').slice() : [];
    const key = authQuotaItemKey(item);
    const provider = String(authQuotaValue(item, 'provider') || '');
    const idx = items.findIndex(existing => authQuotaItemKey(existing) === key && String(authQuotaValue(existing, 'provider') || '') === provider);
    if (idx < 0) return;
    items[idx] = item;
    state.authQuotas = Object.assign({}, current, { items });
  }
  function parseAuthQuotaConcurrency(raw) {
    const text = String(raw || '').trim();
    if (text === '') return 0;
    const value = Number(text);
    if (!Number.isInteger(value) || value < 0) return NaN;
    return value;
  }
  async function saveAuthQuotaConcurrency(input) {
    const itemKey = input.getAttribute('data-item-key') || '';
    const provider = input.getAttribute('data-provider') || '';
    const authID = input.getAttribute('data-auth-id') || '';
    const authIndex = input.getAttribute('data-auth-index') || '';
    const maxConcurrent = parseAuthQuotaConcurrency(input.value);
    if (!Number.isInteger(maxConcurrent) || maxConcurrent < 0) {
      flash('最大并发必须是大于等于 0 的整数', false);
      renderAuthQuotas();
      return;
    }
    const result = await api('POST', 'credit-manager/auth-quotas/concurrency', {
      provider,
      auth_id: authID,
      auth_index: authIndex,
      max_concurrent_requests: maxConcurrent
    });
    const item = authQuotaValue(result, 'item') || result;
    if (item) replaceAuthQuotaItem(item);
    else {
      const current = state.authQuotas && typeof state.authQuotas === 'object' ? state.authQuotas : {};
      const items = Array.isArray(authQuotaValue(current, 'items')) ? authQuotaValue(current, 'items').slice() : [];
      const idx = items.findIndex(existing => authQuotaItemKey(existing) === itemKey);
      if (idx >= 0) items[idx] = Object.assign({}, items[idx], { max_concurrent_requests: maxConcurrent });
      state.authQuotas = Object.assign({}, current, { items });
    }
    renderAuthQuotas();
    flash('认证并发已更新', true);
  }
  async function saveAuthQuotaConcurrencyBatch(scope) {
    const maxConcurrent = parseAuthQuotaConcurrency($('authQuotaBatchConcurrency') && $('authQuotaBatchConcurrency').value);
    if (!Number.isInteger(maxConcurrent) || maxConcurrent < 0) {
      flash('最大并发必须是大于等于 0 的整数', false);
      return;
    }
    const payload = { max_concurrent_requests: maxConcurrent };
    if (state.authQuotaProvider) payload.provider = state.authQuotaProvider;
    const q = String(state.authQuotaName || '').trim();
    if (q) payload.q = q;
    if (scope === 'page') {
      const items = state.authQuotas && Array.isArray(authQuotaValue(state.authQuotas, 'items')) ? authQuotaValue(state.authQuotas, 'items') : [];
      if (!items.length) {
        flash('没有可更新的认证', false);
        return;
      }
      payload.items = items.map(item => ({
        provider: authQuotaValue(item, 'provider') || '',
        auth_id: authQuotaValue(item, 'auth_id') || authQuotaValue(item, 'auth_index') || ''
      })).filter(item => item.auth_id);
      if (!payload.items.length) {
        flash('没有可更新的认证', false);
        return;
      }
    } else {
      const total = Number(authQuotaValue(state.authQuotas, 'total') || 0);
      const label = maxConcurrent > 0 ? String(maxConcurrent) : '不限制';
      if (!confirm('将把当前筛选的 '+total+' 个认证并发设为 '+label+'？')) return;
    }
    const pageBtn = $('btnAuthQuotaBatchPage');
    const filterBtn = $('btnAuthQuotaBatchFilter');
    if (pageBtn) pageBtn.disabled = true;
    if (filterBtn) filterBtn.disabled = true;
    try {
      const result = await api('POST', 'credit-manager/auth-quotas/concurrency/batch', payload);
      await loadAuthQuotas();
      flash('已更新 '+(Number(authQuotaValue(result, 'updated') || 0))+' 个认证的并发', true);
    } finally {
      if (pageBtn) pageBtn.disabled = false;
      if (filterBtn) filterBtn.disabled = false;
    }
  }
  async function refreshAuthQuota(itemKey, provider, authID, authIndex, options) {
    if (!itemKey || state.authQuotaRefreshing[itemKey]) return;
    const seq = state.tabLoadSeq;
    let refreshed = false;
    state.authQuotaRefreshing[itemKey] = true;
    renderAuthQuotas();
    try {
      const result = await api('POST', 'credit-manager/auth-quotas/refresh', { provider, auth_id: authID, auth_index: authIndex });
      if (seq !== state.tabLoadSeq) return;
      const item = authQuotaValue(result, 'item') || result;
      if (item) replaceAuthQuotaItem(item);
      refreshed = true;
    } finally {
      delete state.authQuotaRefreshing[itemKey];
      if (seq === state.tabLoadSeq) renderAuthQuotas();
    }
    // Reconcile with the saved page so a refresh is visible even when the
    // returned item cannot be matched against the currently rendered card.
    if (refreshed && (!options || options.reconcile !== false) && seq === state.tabLoadSeq) await loadAuthQuotas();
    if (refreshed && (!options || !options.silent)) flash('认证额度已刷新', true);
  }
  function renderAuthQuotaPagination(result) {
    const el = $('authQuotaPagination');
    if (!el) return;
    const total = Number(authQuotaValue(result, 'total') || 0);
    const page = Number(authQuotaValue(result, 'page') || state.authQuotaPage || 1);
    const pageSize = Number(authQuotaValue(result, 'page_size') || state.authQuotaPageSize || 12);
    const totalPages = Math.max(Number(authQuotaValue(result, 'total_pages') || 0), 1);
    state.authQuotaPage = page;
    state.authQuotaPageSize = pageSize;
    const start = total ? (page - 1) * pageSize + 1 : 0;
    const end = Math.min(page * pageSize, total);
    el.innerHTML = '<span class="muted"><span>显示</span> '+start+'–'+end+'，<span>共</span> '+total.toLocaleString()+' <span>条</span></span>' +
      '<label>每页<select id="authQuotaPageSize"><option value="8">8 '+t('条')+'</option><option value="12">12 '+t('条')+'</option><option value="16">16 '+t('条')+'</option><option value="24">24 '+t('条')+'</option></select></label>' +
      '<button class="btn ghost" id="btnAuthQuotaPrev" '+(page <= 1 ? 'disabled' : '')+'>'+uiIcon('chevronL')+esc(t('上一页'))+'</button>' +
      '<span class="muted"><span>第</span> '+page+' / '+totalPages+' <span>页</span></span>' +
      '<button class="btn ghost" id="btnAuthQuotaNext" '+(page >= totalPages ? 'disabled' : '')+'>'+esc(t('下一页'))+uiIcon('chevronR')+'</button>';
    $('authQuotaPageSize').value = String(pageSize);
    initCustomControls(el);
    refreshCustomControl($('authQuotaPageSize'));
    $('authQuotaPageSize').addEventListener('change', async event => {
      try {
        state.tabLoadSeq += 1;
        state.authQuotaPageSize = Number(event.target.value);
        state.authQuotaPage = 1;
        await loadAuthQuotas();
      } catch (e) { flash(e.message, false); }
    });
    $('btnAuthQuotaPrev').addEventListener('click', async () => {
      try {
        state.tabLoadSeq += 1;
        state.authQuotaPage = Math.max(1, page - 1);
        await loadAuthQuotas();
      } catch (e) { flash(e.message, false); }
    });
    $('btnAuthQuotaNext').addEventListener('click', async () => {
      try {
        state.tabLoadSeq += 1;
        state.authQuotaPage = Math.min(totalPages, page + 1);
        await loadAuthQuotas();
      } catch (e) { flash(e.message, false); }
    });
  }
  async function refreshVisibleAuthQuotas() {
    if (state.authQuotaPageRefreshing) return;
    const items = state.authQuotas && Array.isArray(authQuotaValue(state.authQuotas, 'items')) ? authQuotaValue(state.authQuotas, 'items').slice() : [];
    if (!items.length) return;
    const seq = ++state.tabLoadSeq;
    const btn = $('btnRefreshAuthQuotaPage');
    state.authQuotaPageRefreshing = true;
    if (btn) btn.disabled = true;
    try {
      for (const item of items) {
        if (seq !== state.tabLoadSeq || state.currentTab !== 'auth-quotas') return;
        try {
          await refreshAuthQuota(
            authQuotaItemKey(item),
            authQuotaValue(item, 'provider') || '',
            authQuotaValue(item, 'auth_id') || '',
            authQuotaValue(item, 'auth_index') || '',
            { silent: true, reconcile: false }
          );
        } catch (_) {}
      }
      if (seq !== state.tabLoadSeq || state.currentTab !== 'auth-quotas') return;
      await loadAuthQuotas();
      flash('本页认证额度已刷新', true);
    } finally {
      state.authQuotaPageRefreshing = false;
      if (btn) btn.disabled = false;
    }
  }
  async function reload() {
    clearFlash();
    state.tabLoadSeq += 1;
    await loadOverviewBundle();
    await loadUsage();
  }

  async function reloadWithModelCatalog() {
    await reload();
    loadModelCatalog().catch(error => {
      const status = $('modelCatalogStatus');
      if (status) status.textContent = '模型目录加载失败：' + (error && error.message ? error.message : error) + '。已连接到 CPA，可稍后重试。';
    });
  }

  function openResetSpendModal(id, all) {
    state.resetSpendAll = Boolean(all);
    state.resetSpendID = all ? '' : id;
    const count = (state.keys || []).length;
    if (all) {
      $('resetSpendTarget').textContent = t('即将重置：') + t('全部启用密钥') + '（' + count + '）';
    } else {
      const key = state.keys.find(item => item.id === id);
      $('resetSpendTarget').textContent = t('即将重置：') + (key && (key.label || key.id) || id);
    }
    $('resetSpendTotal').checked = true;
    $('resetSpendDaily').checked = false;
    $('resetSpendWeekly').checked = false;
    $('resetSpendMonthly').checked = false;
    const modal = $('resetSpendModal');
    modal.classList.add('open');
    modal.setAttribute('aria-hidden', 'false');
    $('btnConfirmResetSpend').focus();
  }

  function closeResetSpendModal() {
    state.resetSpendID = '';
    state.resetSpendAll = false;
    const modal = $('resetSpendModal');
    if (!modal) return;
    modal.classList.remove('open');
    modal.setAttribute('aria-hidden', 'true');
  }

  function selectedSpendResetScopes() {
    return {
      total: Boolean($('resetSpendTotal') && $('resetSpendTotal').checked),
      daily: Boolean($('resetSpendDaily') && $('resetSpendDaily').checked),
      weekly: Boolean($('resetSpendWeekly') && $('resetSpendWeekly').checked),
      monthly: Boolean($('resetSpendMonthly') && $('resetSpendMonthly').checked),
    };
  }

  async function confirmResetKeySpend() {
    const scopes = selectedSpendResetScopes();
    if (!scopes.total && !scopes.daily && !scopes.weekly && !scopes.monthly) {
      flash('请至少选择一项', false);
      return;
    }
    const payload = { ...scopes };
    if (state.resetSpendAll) payload.all = true;
    else payload.id = state.resetSpendID;
    if (!payload.all && !payload.id) return;
    try {
      await api('POST', 'credit-manager/keys/reset-spend', payload);
      closeResetSpendModal();
      await reload();
      flash(t('已用额度已重置'), true);
    } catch (e) { flash(e.message, false); }
  }

  function openDeleteKeyModal(id) {
    const key = state.keys.find(item => item.id === id);
    state.deleteKeyID = id;
    $('deleteKeyTarget').textContent = '即将删除：' + (key && (key.label || key.id) || id);
    const modal = $('deleteKeyModal');
    modal.classList.add('open');
    modal.setAttribute('aria-hidden', 'false');
    $('btnConfirmDeleteKey').focus();
  }

  function closeDeleteKeyModal() {
    state.deleteKeyID = '';
    const modal = $('deleteKeyModal');
    modal.classList.remove('open');
    modal.setAttribute('aria-hidden', 'true');
  }

  async function deleteKeyPermanently(id) {
    try {
      await api('POST', 'credit-manager/keys/delete', { id });
      closeDeleteKeyModal();
      flash('密钥已删除，历史使用统计已保留', true);
      await reload();
    } catch (e) { flash(e.message, false); }
  }

  const closeConnectionModal = () => {
    const modal = $('connectionModal');
    modal.classList.remove('open');
    modal.setAttribute('aria-hidden', 'true');
  };
  const openConnectionModal = () => {
    const modal = $('connectionModal');
    modal.classList.add('open');
    modal.setAttribute('aria-hidden', 'false');
    if (customAPIBaseEnabled()) $('apiBase').focus();
    else $('mgmtToken').focus();
  };

  // events
  document.querySelectorAll('.tab').forEach(btn => btn.addEventListener('click', () => {
    if (btn.dataset.tab === state.currentTab) {
      refreshActiveTab().catch(e => flash(e.message, false));
      return;
    }
    setTab(btn.dataset.tab);
  }));
  document.querySelectorAll('#tokenUnitSwitch [data-token-unit]').forEach(btn => {
    btn.addEventListener('click', () => setTokenUnit(btn.dataset.tokenUnit));
  });
  document.querySelectorAll('#currencySwitch [data-currency]').forEach(btn => {
    btn.addEventListener('click', () => setCurrency(btn.dataset.currency));
  });
  document.querySelectorAll('#modelShareMetricSwitch [data-model-metric]').forEach(btn => {
    btn.addEventListener('click', () => setModelShareMetric(btn.dataset.modelMetric));
  });
  document.querySelectorAll('#modelRankMetricSwitch [data-rank-metric]').forEach(btn => {
    btn.addEventListener('click', () => setModelRankMetric(btn.dataset.rankMetric));
  });
  document.querySelectorAll('.trend-grain-switch [data-trend-grain]').forEach(btn => {
    btn.addEventListener('click', () => setTrendGrain(btn.closest('.trend-grain-switch').dataset.trendChart, btn.dataset.trendGrain));
  });
  window.addEventListener('resize', resizeOverviewCharts);
  $('btnCloseFlash').addEventListener('click', clearFlash);
  $('btnOpenConnection').addEventListener('click', openConnectionModal);
  $('btnCloseConnection').addEventListener('click', closeConnectionModal);
  $('connectionModal').addEventListener('click', event => {
    if (event.target === $('connectionModal')) closeConnectionModal();
  });
  document.addEventListener('keydown', event => {
    if (event.key === 'Escape') {
      closeCustomControls();
      closeKeySearch('overview');
      closeKeySearch('usage');
      closeAuthSearch('overview');
      closeAuthSearch('usage');
      clearFlash();
      closeConnectionModal();
      closeKeyModal();
      closePriceModal();
      closeDeleteKeyModal();
      closeResetSpendModal();
    }
  });
  $('customApiBase').addEventListener('change', () => {
    persistCustomAPIBaseEnabled(customAPIBaseEnabled());
    syncAPIBaseField();
  });
  $('btnSaveToken').addEventListener('click', async () => {
    const t = $('mgmtToken').value.trim();
    if (!t) { flash('请输入管理密钥', false); return; }
    persistCustomAPIBaseEnabled(customAPIBaseEnabled());
    syncAPIBaseField();
    const base = apiBase();
    sessionStorage.setItem(TOKEN_KEY, t);
    sessionStorage.setItem(TOKEN_ORIGIN_KEY, location.origin);
    if (customAPIBaseEnabled() && base) persistCustomAPIBase(base);
    try {
      await reloadWithModelCatalog();
      closeConnectionModal();
      flash('已加载数据', true);
    } catch (e) { flash(e.message, false); }
  });
  $('btnClearToken').addEventListener('click', () => {
    window.clearTimeout(authQuotaSearchTimer);
    sessionStorage.removeItem(TOKEN_KEY);
    sessionStorage.removeItem(TOKEN_ORIGIN_KEY);
    persistCustomAPIBase('');
    $('mgmtToken').value = '';
    syncAPIBaseField();
    clearLoadedSession();
    flash('已清除本地管理密钥与 API 根地址', true);
  });
  let authQuotaSearchTimer = 0;
  $('authQuotaProviderFilter').addEventListener('change', () => {
    state.tabLoadSeq += 1;
    state.authQuotaProvider = $('authQuotaProviderFilter').value || '';
    state.authQuotaPage = 1;
    loadAuthQuotas().catch(e => flash(e.message, false));
  });
  $('authQuotaNameFilter').addEventListener('input', () => {
    state.tabLoadSeq += 1;
    state.authQuotaName = $('authQuotaNameFilter').value || '';
    state.authQuotaPage = 1;
    window.clearTimeout(authQuotaSearchTimer);
    authQuotaSearchTimer = window.setTimeout(() => {
      loadAuthQuotas().catch(e => flash(e.message, false));
    }, 300);
  });
  $('btnRefreshAuthQuotaPage').addEventListener('click', () => {
    refreshVisibleAuthQuotas().catch(e => flash(e.message, false));
  });
  $('btnAuthQuotaBatchPage').addEventListener('click', () => {
    saveAuthQuotaConcurrencyBatch('page').catch(e => flash(e.message, false));
  });
  $('btnAuthQuotaBatchFilter').addEventListener('click', () => {
    saveAuthQuotaConcurrencyBatch('filter').catch(e => flash(e.message, false));
  });
  $('authQuotaList').addEventListener('change', event => {
    const select = event.target.closest('.auth-quota-week-select');
    if (select) {
      const authID = select.getAttribute('data-auth-id') || '';
      if (!authID) return;
      state.authQuotaWeeks[authID] = select.value || '';
      renderAuthQuotas();
      return;
    }
    const input = event.target.closest('.auth-quota-concurrency-input');
    if (!input) return;
    saveAuthQuotaConcurrency(input).catch(e => flash(e.message, false));
  });
  $('authQuotaList').addEventListener('click', async event => {
    const button = event.target.closest('.auth-quota-reload');
    if (!button) return;
    try {
      await refreshAuthQuota(
        button.getAttribute('data-item-key') || '',
        button.getAttribute('data-provider') || '',
        button.getAttribute('data-auth-id') || '',
        button.getAttribute('data-auth-index') || ''
      );
    } catch (e) { flash(e.message, false); }
  });

  $('btnRefresh').addEventListener('click', async () => {
    try {
      if (state.currentTab === 'auth-quotas') { await loadAuthQuotas(); flash('认证额度已从缓存刷新', true); }
      else { await reloadWithModelCatalog(); flash('数据已刷新', true); }
    } catch (e) { flash(e.message, false); }
  });
  $('overviewRangeFilter').addEventListener('change', setOverviewRangeVisibility);
  $('usageRangeFilter').addEventListener('change', setUsageRangeVisibility);
  ['overview', 'usage'].forEach(kind => {
    const input = $(kind + 'KeyFilter');
    input.addEventListener('focus', () => openKeySearch(kind));
    input.addEventListener('input', () => {
      delete input.dataset.selectedKeyId;
      renderKeySearchOptions(kind);
      openKeySearch(kind);
    });
    input.addEventListener('keydown', event => {
      if (event.key === 'Escape') {
        closeKeySearch(kind);
        input.blur();
      }
      if (event.key === 'Enter') closeKeySearch(kind);
    });
    const authInput = $(kind + 'AuthFilter');
    authInput.addEventListener('focus', () => openAuthSearch(kind));
    authInput.addEventListener('input', () => {
      renderAuthSearchOptions(kind);
      openAuthSearch(kind);
    });
    authInput.addEventListener('keydown', event => {
      if (event.key === 'Escape') {
        closeAuthSearch(kind);
        authInput.blur();
      }
      if (event.key === 'Enter') closeAuthSearch(kind);
    });
  });
  $('btnLoadOverview').addEventListener('click', async () => {
    try {
      await reload();
      flash('概览已按当前条件刷新', true);
    } catch (e) { flash(e.message, false); }
  });
  $('btnResetOverview').addEventListener('click', async () => {
    $('overviewRangeFilter').value = 'today';
    $('overviewKeyFilter').value = '';
    delete $('overviewKeyFilter').dataset.selectedKeyId;
    $('overviewAuthFilter').value = '';
    closeKeySearch('overview');
    closeAuthSearch('overview');
    $('overviewModelFilter').value = '';
    $('overviewSourceFilter').value = '';
    $('overviewFromFilter').value = '';
    $('overviewToFilter').value = '';
    setTrendGrain('token', 'day');
    setTrendGrain('cost', 'day');
    refreshCustomControls();
    setOverviewRangeVisibility();
    try {
      await reload();
      flash('概览筛选已重置', true);
    } catch (e) { flash(e.message, false); }
  });
  $('btnLoadModelCatalog').addEventListener('click', async () => {
    try { await loadModelCatalog(); flash(state.modelCatalogError ? '已加载代理模型，价格目录暂不可用' : '已加载全部模型并同步新价格规则', true); } catch (e) { $('modelCatalogStatus').textContent = '加载失败：' + e.message; flash(e.message, false); }
  });
  $('btnOpenCreateKey').addEventListener('click', () => openKeyModal('create'));
  $('btnResetAllKeySpend').addEventListener('click', () => {
    if (!(state.keys || []).length) return;
    openResetSpendModal('', true);
  });
  $('btnResetManagedKeySpend').addEventListener('click', () => {
    const id = $('keyModalId').value;
    if (!id) return;
    openResetSpendModal(id);
  });
  $('btnCloseResetSpendModal').addEventListener('click', closeResetSpendModal);
  $('btnCancelResetSpend').addEventListener('click', closeResetSpendModal);
  $('resetSpendModal').addEventListener('click', event => {
    if (event.target === $('resetSpendModal')) closeResetSpendModal();
  });
  $('btnConfirmResetSpend').addEventListener('click', confirmResetKeySpend);
  $('btnClosePriceModal').addEventListener('click', closePriceModal);
  $('btnCancelPriceModal').addEventListener('click', closePriceModal);
  $('priceModal').addEventListener('click', event => {
    if (event.target === $('priceModal')) closePriceModal();
  });
  $('btnCloseKeyModal').addEventListener('click', closeKeyModal);
  $('btnCancelKeyModal').addEventListener('click', closeKeyModal);
  $('keyModal').addEventListener('click', event => {
    if (event.target === $('keyModal')) closeKeyModal();
  });
  $('btnCloseDeleteKeyModal').addEventListener('click', closeDeleteKeyModal);
  $('btnCancelDeleteKey').addEventListener('click', closeDeleteKeyModal);
  $('deleteKeyModal').addEventListener('click', event => {
    if (event.target === $('deleteKeyModal')) closeDeleteKeyModal();
  });
  $('btnConfirmDeleteKey').addEventListener('click', async () => {
    if (!state.deleteKeyID) return;
    await deleteKeyPermanently(state.deleteKeyID);
  });
  $('keyModalTokenLimitsEnabled').addEventListener('change', event => {
    setTokenLimitsSectionVisible(event.target.checked, event.target.disabled);
  });
  $('btnAddKeyTokenLimit').addEventListener('click', () => addKeyTokenLimit($('keyModalTokenLimitModel').value));
  $('keyModalTokenLimitModel').addEventListener('focus', openTokenLimitModelSearch);
  $('keyModalTokenLimitModel').addEventListener('input', openTokenLimitModelSearch);
  $('keyModalTokenLimitModel').addEventListener('keydown', event => {
    if (event.key === 'Enter') {
      event.preventDefault();
      addKeyTokenLimit($('keyModalTokenLimitModel').value);
    }
    if (event.key === 'Escape') {
      event.preventDefault();
      closeTokenLimitModelSearch();
    }
    if (event.key === 'ArrowDown') openTokenLimitModelSearch();
  });
  $('keyModal').querySelector('.modal-body').addEventListener('scroll', closeTokenLimitModelSearch, { passive:true });
  window.addEventListener('resize', closeTokenLimitModelSearch);
  $('keyModalUnmatched').addEventListener('click', event => {
    const button = event.target.closest('[data-unmatched-set]');
    if (!button || button.disabled) return;
    setUnmatchedModelsMode(button.dataset.unmatchedSet);
  });
  $('keyModalTokenLimits').addEventListener('click', event => {
    const modeButton = event.target.closest('[data-mode-set]');
    if (modeButton && !modeButton.disabled) {
      const period = modeButton.closest('.token-limit-period');
      period.dataset.mode = modeButton.dataset.modeSet;
      syncTokenLimitPeriod(period);
      return;
    }
    const button = event.target.closest('[data-remove-token-limit]');
    if (!button || button.disabled) return;
    renderKeyTokenLimits(collectModelTokenLimits().filter(item => item.model !== button.dataset.removeTokenLimit), $('keyModalMode').value === 'rotate');
  });
  $('keyModalTokenLimits').addEventListener('input', event => {
    const period = event.target.closest('.token-limit-period');
    if (period) syncTokenLimitPeriod(period);
  });
  $('btnGenerateKeyMaterial').addEventListener('click', () => {
    $('keyMaterial').value = createKeyMaterialSuffix();
    $('keyMaterial').focus();
  });
  $('btnRevealManagedKey').addEventListener('click', () => {
    const id = $('keyModalId').value;
    if (id) revealKey(id);
  });
  $('btnRotateManagedKey').addEventListener('click', () => {
    const id = $('keyModalId').value;
    if (id) openKeyModal('rotate', id);
  });
  $('btnCopyKeyPlaintext').addEventListener('click', async () => {
    try {
      await copyText($('keyPlaintext').textContent || '');
      flash('已复制密钥', true);
    } catch (_) { flash('复制失败，请手动复制', false); }
  });
  $('btnSubmitKeyModal').addEventListener('click', async () => {
    try { await submitKeyModal(); } catch (e) { flash(e.message, false); }
  });
  $('btnLoadModelPrices').addEventListener('click', async () => {
    try { await loadModelPrices(); } catch (e) { $('modelPriceStatus').textContent = '同步失败：' + e.message; flash(e.message, false); }
  });
  $('priceModelPicker').addEventListener('change', event => loadModelPrice(event.target.value));
  $('priceBillingMode').addEventListener('change', syncPriceBillingFields);
  $('btnAddContextTier').addEventListener('click', () => {
    $('priceContextTiers').insertAdjacentHTML('beforeend', contextTierCardHTML({}));
  });
  $('priceContextTiers').addEventListener('click', event => {
    const button = event.target.closest('[data-remove-context-tier]');
    if (button) button.closest('.price-tier-row').remove();
  });
  $('btnSavePrice').addEventListener('click', async () => {
    try {
      const body = {
        id: $('priceId').value.trim(),
        match_kind: $('priceKind').value,
        pattern: $('pricePattern').value.trim(),
        priority: Number($('pricePriority').value || 0),
        enabled: $('priceEnabled').checked,
        price: {
          input: microFromUSD($('priceIn').value) || 0,
          output: microFromUSD($('priceOut').value) || 0,
          cache_read: microFromUSD($('priceCacheRead').value) || 0,
          cache_creation: microFromUSD($('priceCacheCreation').value) || 0,
          accounting_mode: $('priceAccountingMode').value || '',
          billing_mode: $('priceBillingMode').value || 'token',
          per_image: microFromUSD($('pricePerImage').value) || 0,
        },
        tiers: $('priceBillingMode').value === 'per_image' ? [] : collectPriceTiers(),
      };
      if (!body.id || !body.pattern) throw new Error('规则 ID 与 pattern 必填');
      await api('POST', 'credit-manager/pricing', body);
      closePriceModal();
      flash('价格规则已保存', true);
      await reload();
    } catch (e) { flash(e.message, false); }
  });
  $('btnLoadUsage').addEventListener('click', async () => {
    try {
      await loadUsage(true);
      flash('统计已按当前条件刷新', true);
    } catch (e) { flash(e.message, false); }
  });
  $('btnClearUsageFilters').addEventListener('click', async () => {
    $('usageRangeFilter').value = 'today';
    ['usageKeyFilter', 'usageAuthFilter', 'usageModelFilter', 'usageSourceFilter', 'usageFromFilter', 'usageToFilter', 'usageMinCost', 'usageMaxCost', 'usageMinTokens', 'usageMaxTokens'].forEach(id => { $(id).value = ''; });
    closeKeySearch('usage');
    closeAuthSearch('usage');
    refreshCustomControls();
    setUsageRangeVisibility();
    try {
      await loadUsage(true);
      flash('统计筛选已清除', true);
    } catch (e) { flash(e.message, false); }
  });

  // boot
  restoreCustomAPIBasePreference();
  syncAPIBaseField();
  const savedBase = detectDefaultBase();
  state.tokenUnit = normalizeTokenUnit(localStorage.getItem(TOKEN_UNIT_KEY) || 'raw');
  state.currency = normalizeCurrency(localStorage.getItem(CURRENCY_KEY) || 'USD');
  state.usdCnyRate = normalizeUsdCnyRate(localStorage.getItem(USD_CNY_RATE_KEY) || DEFAULT_USD_CNY_RATE);
  syncTokenUnitSwitch();
  syncCurrencySwitch();
  syncModelShareMetricSwitch();
  initCustomControls();
  initPreferences();
  fetchUsdCnyRate(false);
  window.setInterval(() => fetchUsdCnyRate(false), 30 * 60 * 1000);
  if (savedBase) $('apiBase').value = savedBase;
  setOverviewRangeVisibility();
  setUsageRangeVisibility();
  const saved = savedSessionToken();
  if (saved && (customAPIBaseEnabled() || isSameOriginBase(apiBase()))) {
    $('mgmtToken').value = saved;
    reloadWithModelCatalog().catch(e => flash(e.message, false));
  } else {
    clearLoadedSession();
  }
})();
