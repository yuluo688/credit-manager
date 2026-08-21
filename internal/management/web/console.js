
(function () {
  const TOKEN_KEY = 'credit_manager_mgmt_token';
  const BASE_KEY = 'credit_manager_api_base';
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
    '概览': { 'zh-TW':'概覽', en:'Overview', ru:'Обзор' }, 'Key 管理': { 'zh-TW':'Key 管理', en:'Keys', ru:'Ключи' }, '模型与价格': { 'zh-TW':'模型與價格', en:'Models & pricing', ru:'Модели и цены' }, '使用统计': { 'zh-TW':'使用統計', en:'Usage analytics', ru:'Статистика использования' },
    '额度概览': { 'zh-TW':'額度概覽', en:'Credit overview', ru:'Обзор лимитов' }, '按时间、Key、账号、模型和来源聚合账本数据。': { 'zh-TW':'依時間、Key、帳號、模型與來源彙總帳本資料。', en:'Aggregate ledger data by time, key, account, model, and source.', ru:'Агрегируйте данные журнала по времени, ключу, аккаунту, модели и источнику.' }, '账本运行中': { 'zh-TW':'帳本運行中', en:'Ledger online', ru:'Журнал работает' },
    '概览筛选': { 'zh-TW':'概覽篩選', en:'Overview filters', ru:'Фильтры обзора' }, '筛选仅影响概览指标和图表；「今日」按本地自然日。': { 'zh-TW':'篩選僅影響概覽指標和圖表；「今天」按本地自然日。', en:'Filters affect overview metrics and charts only; Today uses the local calendar day.', ru:'Фильтры влияют только на метрики и диаграммы; «Сегодня» — локальный день.' }, '今日 · 全部数据': { 'zh-TW':'今天 · 全部資料', en:'Today · all data', ru:'Сегодня · все данные' }, '最近 30 天 · 全部数据': { 'zh-TW':'最近 30 天 · 全部資料', en:'Last 30 days · all data', ru:'Последние 30 дней · все данные' },
    '时间范围': { 'zh-TW':'時間範圍', en:'Time range', ru:'Период' }, '今日': { 'zh-TW':'今天', en:'Today', ru:'Сегодня' }, '最近 7 天': { 'zh-TW':'最近 7 天', en:'Last 7 days', ru:'Последние 7 дней' }, '最近 30 天': { 'zh-TW':'最近 30 天', en:'Last 30 days', ru:'Последние 30 дней' }, '最近 90 天': { 'zh-TW':'最近 90 天', en:'Last 90 days', ru:'Последние 90 дней' }, '全部时间': { 'zh-TW':'全部時間', en:'All time', ru:'За всё время' }, '自定义范围': { 'zh-TW':'自訂範圍', en:'Custom range', ru:'Свой период' }, '时间范围（UTC）': { 'zh-TW':'時間範圍（UTC）', en:'Time range (UTC)', ru:'Период (UTC)' }, '至': { 'zh-TW':'至', en:'to', ru:'до' },
    'Key': { 'zh-TW':'Key', en:'Key', ru:'Ключ' }, '全部 Key': { 'zh-TW':'全部 Key', en:'All keys', ru:'Все ключи' }, '账号': { 'zh-TW':'帳號', en:'Account', ru:'Аккаунт' }, '搜索账号名称或 ID': { 'zh-TW':'搜尋帳號名稱或 ID', en:'Search account name or ID', ru:'Поиск аккаунта или ID' }, '模型': { 'zh-TW':'模型', en:'Model', ru:'Модель' }, '全部已使用模型': { 'zh-TW':'全部已使用模型', en:'All used models', ru:'Все использованные модели' }, '来源': { 'zh-TW':'來源', en:'Source', ru:'Источник' }, '全部来源': { 'zh-TW':'全部來源', en:'All sources', ru:'Все источники' },
    '重置': { 'zh-TW':'重設', en:'Reset', ru:'Сбросить' }, '刷新概览': { 'zh-TW':'重新整理概覽', en:'Refresh overview', ru:'Обновить обзор' }, 'Token 趋势': { 'zh-TW':'Token 趨勢', en:'Token trend', ru:'Динамика токенов' }, '费用趋势': { 'zh-TW':'費用趨勢', en:'Cost trend', ru:'Динамика расходов' },     '模型调用占比': { 'zh-TW':'模型呼叫占比', en:'Model usage share', ru:'Доля вызовов моделей' }, '模型效率排行': { 'zh-TW':'模型效率排行', en:'Model efficiency rank', ru:'Рейтинг эффективности' }, '模型效率指标': { 'zh-TW':'模型效率指標', en:'Efficiency metric', ru:'Метрика эффективности' }, '性价比': { 'zh-TW':'性價比', en:'Value', ru:'Выгода' }, '单次': { 'zh-TW':'單次', en:'Per call', ru:'За запрос' }, '吞吐': { 'zh-TW':'吞吐', en:'Throughput', ru:'Скорость' },     '按每美元产出 Token 排序，越高越省': { 'zh-TW':'依每美元產出 Token 排序，越高越省', en:'Ranked by tokens per dollar; higher is cheaper', ru:'По токенам на доллар; больше — выгоднее' }, '按每人民币产出 Token 排序，越高越省': { 'zh-TW':'依每人民幣產出 Token 排序，越高越省', en:'Ranked by tokens per yuan; higher is cheaper', ru:'По токенам на юань; больше — выгоднее' }, '平均每次请求费用，越低越省': { 'zh-TW':'平均每次請求費用，越低越省', en:'Average cost per request; lower is cheaper', ru:'Средняя цена запроса; меньше — выгоднее' }, '缓存读取占输入比例，越高越省': { 'zh-TW':'快取讀取佔輸入比例，越高越省', en:'Cache-read share of input; higher is cheaper', ru:'Доля чтения кэша во входе; больше — выгоднее' }, '平均生成速度，越高越快': { 'zh-TW':'平均生成速度，越高越快', en:'Average generation speed; higher is faster', ru:'Средняя скорость генерации; больше — быстрее' }, '当前筛选条件下暂无模型效率数据': { 'zh-TW':'目前篩選條件下暫無模型效率資料', en:'No model efficiency data for the current filters', ru:'Нет данных об эффективности моделей' }, '免费': { 'zh-TW':'免費', en:'Free', ru:'Бесплатно' }, '次': { 'zh-TW':'次', en:'calls', ru:'вызов.' }, '调用次数': { 'zh-TW':'呼叫次數', en:'Requests', ru:'Запросы' }, '占比': { 'zh-TW':'佔比', en:'Share', ru:'Доля' }, '费用': { 'zh-TW':'費用', en:'Cost', ru:'Стоимость' }, '入': { 'zh-TW':'入', en:'In', ru:'Вх.' }, '出': { 'zh-TW':'出', en:'Out', ru:'Исх.' }, '缓存': { 'zh-TW':'快取', en:'Cache', ru:'Кэш' }, '命中': { 'zh-TW':'命中', en:'Hit', ru:'Попад.' }, '未连接': { 'zh-TW':'未連線', en:'Disconnected', ru:'Нет связи' },
    '趋势时间维度': { 'zh-TW':'趨勢時間維度', en:'Trend interval', ru:'Интервал тренда' }, 'Token 趋势时间维度': { 'zh-TW':'Token 趨勢時間維度', en:'Token trend interval', ru:'Интервал токенов' }, '费用趋势时间维度': { 'zh-TW':'費用趨勢時間維度', en:'Cost trend interval', ru:'Интервал расходов' }, '时': { 'zh-TW':'時', en:'Hour', ru:'Час' }, '日': { 'zh-TW':'日', en:'Day', ru:'День' }, '月': { 'zh-TW':'月', en:'Month', ru:'Месяц' },
    '按 Key 设置额度、启停状态和可用模型策略。': { 'zh-TW':'依 Key 設定額度、啟停狀態和可用模型策略。', en:'Set limits, status, and model access for each key.', ru:'Настройте лимиты, статус и доступ к моделям для каждого ключа.' }, '额度隔离': { 'zh-TW':'額度隔離', en:'Isolated limits', ru:'Изолированные лимиты' }, '添加 Key': { 'zh-TW':'新增 Key', en:'Add key', ru:'Добавить ключ' }, '已有 Key': { 'zh-TW':'已有 Key', en:'Existing keys', ru:'Существующие ключи' }, '额度与状态一览': { 'zh-TW':'額度與狀態一覽', en:'Limits and status', ru:'Лимиты и статус' },
    '模型与价格': { 'zh-TW':'模型與價格', en:'Models & pricing', ru:'Модели и цены' }, '文本模型按百万 Token 计价；纯出图模型按张计费，不能套用 Token 价。': { 'zh-TW':'文字模型按百萬 Token 計價；純出圖模型按張計費，不能套用 Token 價。', en:'Text models bill per million tokens; image models bill per image.', ru:'Текстовые модели тарифицируются за миллион токенов, генерация изображений — за картинку.' }, '定价规则': { 'zh-TW':'定價規則', en:'Pricing rules', ru:'Правила цен' }, '当前代理模型': { 'zh-TW':'目前代理模型', en:'Current proxy models', ru:'Текущие модели прокси' }, '加载全部模型': { 'zh-TW':'載入全部模型', en:'Load all models', ru:'Загрузить все модели' }, '加载当前代理公开的模型后，可设置价格，或启用/禁用单个模型。禁用后无法调用，也不会出现在客户端模型列表中。': { 'zh-TW':'載入目前代理公開的模型後，可設定價格，或啟用/停用單一模型。停用後無法呼叫，也不會出現在客戶端模型列表中。', en:'After loading proxy models, set prices or enable/disable a model. Disabled models cannot be called and are omitted from the client model list.', ru:'После загрузки моделей задайте цены или включите/отключите модель. Отключённые модели нельзя вызвать, и они не попадают в клиентский список.' },     '启用': { 'zh-TW':'啟用', en:'Enable', ru:'Включить' }, '禁用': { 'zh-TW':'停用', en:'Disable', ru:'Отключить' }, '状态': { 'zh-TW':'狀態', en:'Status', ru:'Статус' }, '模型已启用': { 'zh-TW':'模型已啟用', en:'Model enabled', ru:'Модель включена' }, '模型已禁用': { 'zh-TW':'模型已停用', en:'Model disabled', ru:'Модель отключена' },
    '计费方式': { 'zh-TW':'計費方式', en:'Billing mode', ru:'Режим тарифа' }, '按 Token（USD / 1M）': { 'zh-TW':'按 Token（USD / 1M）', en:'Per token (USD / 1M)', ru:'За токен (USD / 1M)' }, '按张（出图）': { 'zh-TW':'按張（出圖）', en:'Per image', ru:'За изображение' }, '每张 USD': { 'zh-TW':'每張 USD', en:'USD / image', ru:'USD / изображение' }, '出图': { 'zh-TW':'出圖', en:'Image', ru:'Картинка' }, '按张计费': { 'zh-TW':'按張計費', en:'Per image', ru:'За картинку' },
    '从请求、Token 到费用的可筛选账本视图。': { 'zh-TW':'從請求、Token 到費用的可篩選帳本檢視。', en:'A filterable ledger view from requests and tokens to costs.', ru:'Фильтруемый журнал от запросов и токенов до расходов.' }, '实时汇总': { 'zh-TW':'即時彙總', en:'Live summary', ru:'Сводка в реальном времени' }, '统计筛选': { 'zh-TW':'統計篩選', en:'Usage filters', ru:'Фильтры статистики' }, '应用筛选': { 'zh-TW':'套用篩選', en:'Apply filters', ru:'Применить фильтры' }, '清除筛选': { 'zh-TW':'清除篩選', en:'Clear filters', ru:'Очистить фильтры' }, '按 Key 汇总': { 'zh-TW':'依 Key 彙總', en:'By key', ru:'По ключам' }, '按模型汇总': { 'zh-TW':'依模型彙總', en:'By model', ru:'По моделям' }, '最近明细': { 'zh-TW':'最近明細', en:'Recent activity', ru:'Последние записи' },
    '关闭提示': { 'zh-TW':'關閉提示', en:'Close notification', ru:'Закрыть уведомление' }, '取消': { 'zh-TW':'取消', en:'Cancel', ru:'Отмена' }, '保存规则': { 'zh-TW':'儲存規則', en:'Save rule', ru:'Сохранить правило' }, '删除': { 'zh-TW':'刪除', en:'Delete', ru:'Удалить' }, '编辑': { 'zh-TW':'編輯', en:'Edit', ru:'Изменить' }, '复制 Key': { 'zh-TW':'複製 Key', en:'Copy key', ru:'Копировать ключ' }, '管理 Key': { 'zh-TW':'管理 Key', en:'Manage key', ru:'Управлять ключом' },
    'CLIProxyAPI 根地址': { 'zh-TW':'CLIProxyAPI 根位址', en:'CLIProxyAPI base URL', ru:'Базовый URL CLIProxyAPI' }, '宿主管理密钥（Bearer）': { 'zh-TW':'宿主管理金鑰（Bearer）', en:'Host management token (Bearer)', ru:'Токен управления хостом (Bearer)' }, '清除本地信息': { 'zh-TW':'清除本機資訊', en:'Clear local data', ru:'Очистить локальные данные' }, '连接并加载': { 'zh-TW':'連線並載入', en:'Connect and load', ru:'Подключиться и загрузить' },
    '创建 Key': { 'zh-TW':'建立 Key', en:'Create key', ru:'Создать ключ' }, '保存策略': { 'zh-TW':'儲存策略', en:'Save policy', ru:'Сохранить политику' }, '确认轮换': { 'zh-TW':'確認輪換', en:'Confirm rotation', ru:'Подтвердить ротацию' }, '新增价格规则': { 'zh-TW':'新增價格規則', en:'Add pricing rule', ru:'Добавить правило цены' }, '编辑价格规则': { 'zh-TW':'編輯價格規則', en:'Edit pricing rule', ru:'Изменить правило цены' },
    '总额度（USD）': { 'zh-TW':'總額度（USD）', en:'Total quota (USD)', ru:'Общий лимит (USD)' }, '日额度（USD）': { 'zh-TW':'日額度（USD）', en:'Daily quota (USD)', ru:'Дневной лимит (USD)' }, '周额度（USD）': { 'zh-TW':'週額度（USD）', en:'Weekly quota (USD)', ru:'Недельный лимит (USD)' }, '月额度（USD）': { 'zh-TW':'月額度（USD）', en:'Monthly quota (USD)', ru:'Месячный лимит (USD)' }, '最大并发请求数': { 'zh-TW':'最大併發請求數', en:'Max concurrent requests', ru:'Макс. параллельных запросов' },
    '认证额度': { 'zh-TW':'認證額度', en:'Auth quotas', ru:'Квоты авторизации' }, '每个认证同时显示 5 小时窗口和当前额度周，可在卡片内切换该账号的其他额度周。刷新最多每 15 分钟查询一次。': { 'zh-TW':'每個認證同時顯示 5 小時窗口與目前額度週，可在卡片內切換該帳號的其他額度週。重新整理最多每 15 分鐘查詢一次。', en:'Each auth shows its 5-hour window and current quota week. Switch weeks inside the card. Refresh is limited to once every 15 minutes.', ru:'У каждой авторизации видно 5-часовое окно и текущую неделю квоты; неделю можно сменить в карточке. Обновление не чаще одного раза в 15 минут.' }, '额度周': { 'zh-TW':'額度週', en:'Quota week', ru:'Неделя квоты' }, '5 小时': { 'zh-TW':'5 小時', en:'5 hours', ru:'5 часов' }, '周额度': { 'zh-TW':'週額度', en:'Weekly', ru:'Неделя' }, '当前费用': { 'zh-TW':'目前費用', en:'Used cost', ru:'Текущие расходы' }, '预估剩余': { 'zh-TW':'預估剩餘', en:'Est. remaining', ru:'Ост. расходы' }, '预计可用': { 'zh-TW':'預計可用', en:'Est. available', ru:'Доступно' }, '平台': { 'zh-TW':'平台', en:'Platform', ru:'Платформа' }, '全部平台': { 'zh-TW':'全部平台', en:'All platforms', ru:'Все платформы' }, '名称': { 'zh-TW':'名稱', en:'Name', ru:'Имя' }, '搜索账号或名称': { 'zh-TW':'搜尋帳號或名稱', en:'Search account or name', ru:'Поиск аккаунта или имени' },
    'CPA 额度管理': { 'zh-TW':'CPA 額度管理', en:'CPA Credit Manager', ru:'Менеджер лимитов CPA' }, '选择日期和时间': { 'zh-TW':'選擇日期和時間', en:'Select date and time', ru:'Выберите дату и время' }, '上个月': { 'zh-TW':'上個月', en:'Previous month', ru:'Предыдущий месяц' }, '下个月': { 'zh-TW':'下個月', en:'Next month', ru:'Следующий месяц' }, '减少小时': { 'zh-TW':'減少小時', en:'Decrease hours', ru:'Уменьшить часы' }, '减少分钟': { 'zh-TW':'減少分鐘', en:'Decrease minutes', ru:'Уменьшить минуты' }, '增加小时': { 'zh-TW':'增加小時', en:'Increase hours', ru:'Увеличить часы' }, '增加分钟': { 'zh-TW':'增加分鐘', en:'Increase minutes', ru:'Увеличить минуты' }, '时间': { 'zh-TW':'時間', en:'Time', ru:'Время' }, '清除': { 'zh-TW':'清除', en:'Clear', ru:'Очистить' }, '此刻': { 'zh-TW':'此刻', en:'Now', ru:'Сейчас' },
    'Token 显示单位': { 'zh-TW':'Token 顯示單位', en:'Token display unit', ru:'Единица отображения токенов' }, '原始数量': { 'zh-TW':'原始數量', en:'Raw count', ru:'Исходное количество' },
    '千 (×1,000)': { 'zh-TW':'千 (×1,000)', en:'Thousand (×1,000)', ru:'Тысячи (×1,000)' }, 'k (×1,000)': { 'zh-TW':'k (×1,000)', en:'k (×1,000)', ru:'k (×1,000)' }, '万 (×10,000)': { 'zh-TW':'萬 (×10,000)', en:'10 thousand (×10,000)', ru:'Десятки тысяч (×10,000)' }, 'w (×10,000)': { 'zh-TW':'w (×10,000)', en:'w (×10,000)', ru:'w (×10,000)' }, '百万 (×1,000,000)': { 'zh-TW':'百萬 (×1,000,000)', en:'Million (×1,000,000)', ru:'Миллионы (×1,000,000)' }, 'm (×1,000,000)': { 'zh-TW':'m (×1,000,000)', en:'m (×1,000,000)', ru:'m (×1,000,000)' },
    '个': { 'zh-TW':'個', en:'个', ru:'个' }, '千': { 'zh-TW':'千', en:'千', ru:'千' }, 'k': { 'zh-TW':'k', en:'k', ru:'k' }, '万': { 'zh-TW':'萬', en:'万', ru:'万' }, 'w': { 'zh-TW':'w', en:'w', ru:'w' }, '百万': { 'zh-TW':'百萬', en:'百万', ru:'百万' }, 'm': { 'zh-TW':'m', en:'m', ru:'m' },
    '实时美元兑人民币汇率': { 'zh-TW':'即時美元兌人民幣匯率', en:'Live USD to CNY rate', ru:'Курс USD к CNY' },
    '汇率获取失败，已使用上次汇率': { 'zh-TW':'匯率取得失敗，已使用上次匯率', en:'Could not refresh the rate; using the last saved value', ru:'Не удалось обновить курс; используется прошлое значение' },
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
    authQuotaProvider: '',
    authQuotaName: '',
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

  function t(source) {
    const locale = state.locale || 'zh-CN';
    return locale === 'zh-CN' ? source : ((COPY[source] && COPY[source][locale]) || source);
  }

  function translateElement(element) {
    if (!element || element.closest('[data-no-i18n]')) return;
    const attrs = ['title', 'placeholder', 'aria-label'];
    attrs.forEach(name => {
      if (!element.hasAttribute(name)) return;
      let values = attributeSources.get(element);
      if (!values) { values = {}; attributeSources.set(element, values); }
      if (!(name in values)) values[name] = element.getAttribute(name);
      element.setAttribute(name, t(values[name]));
    });
    const textNodes = [...element.childNodes].filter(node => node.nodeType === Node.TEXT_NODE && node.nodeValue.trim());
    if (textNodes.length) {
      let values = textSources.get(element);
      if (!values) { values = new Map(); textSources.set(element, values); }
      textNodes.forEach(node => {
        if (!values.has(node)) values.set(node, node.nodeValue);
        const source = values.get(node);
        const leading = source.match(/^\s*/)[0];
        const trailing = source.match(/\s*$/)[0];
        node.nodeValue = leading + t(source.trim()) + trailing;
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
    text.textContent = msg == null ? '' : String(msg);
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

  function token() {
    return ($('mgmtToken').value || sessionStorage.getItem(TOKEN_KEY) || '').trim();
  }

  function detectDefaultBase() {
    try {
      const q = new URLSearchParams(location.search || '');
      const fromQuery = (q.get('api_base') || q.get('api') || '').trim();
      if (fromQuery) return fromQuery.replace(/\/$/, '');
    } catch (_) {}
    // Resource page served by CLIProxyAPI itself.
    if ((location.pathname || '').indexOf('/v0/resource/plugins/') === 0) {
      return location.origin;
    }
    return (sessionStorage.getItem(BASE_KEY) || '').trim().replace(/\/$/, '');
  }

  function apiBase() {
    const raw = ($('apiBase').value || sessionStorage.getItem(BASE_KEY) || '').trim().replace(/\/$/, '');
    return raw;
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
    const res = await fetch(hostURL(path), { headers: { 'Accept': 'application/json' } });
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
    if (fromManagement.length) {
      return { data: fromManagement.map(id => ({ id })), source: 'management' };
    }
    // Fallback only when management discovery returned nothing.
    try {
      const payload = await hostAPI('v1/models');
      return { data: modelIDs(payload).map(id => ({ id })), source: 'v1/models' };
    } catch (e) {
      const msg = e && e.message ? e.message : String(e);
      if (/missing api key|invalid api key|unauthorized|401/i.test(msg)) {
        throw new Error('无法通过管理接口发现模型，且 /v1/models 鉴权失败（' + msg + '）。请确认宿主已配置上游认证文件。');
      }
      throw e;
    }
  }

  async function fetchModelsViaManagement() {
    const filesPayload = await hostManagementGET('auth-files');
    const files = (filesPayload && filesPayload.files) || [];
    const names = [...new Set(files.map(file => {
      if (!file || typeof file !== 'object') return '';
      return String(file.name || file.id || file.Name || file.ID || '').trim();
    }).filter(Boolean))];
    if (!names.length) return [];
    const ids = new Set();
    const results = await Promise.all(names.map(async name => {
      try {
        return await hostManagementGET('auth-files/models?name=' + encodeURIComponent(name));
      } catch (_) {
        return null;
      }
    }));
    results.forEach(payload => {
      const models = (payload && payload.models) || [];
      models.forEach(model => {
        const id = typeof model === 'string' ? model : (model && (model.id || model.ID || model.name));
        if (id) ids.add(String(id).trim());
      });
    });
    return [...ids].filter(Boolean).sort();
  }

  async function api(method, path, body) {
    const t = token();
    if (!t) throw new Error('请先填写并保存宿主管理密钥');
    const url = managementURL(path);
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
      return chosen.length ? chosen.join('、') : '全部可用';
    }
    return select.selectedOptions[0] ? select.selectedOptions[0].textContent.trim() : '请选择';
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
    if (!event.target.closest('.key-search-control')) {
      closeKeySearch('overview');
      closeKeySearch('usage');
      closeAuthSearch('overview');
      closeAuthSearch('usage');
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
    const data = await api('GET', 'credit-manager/overview' + overviewQuery());
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
      plugin_key_id: resolveKeyFilter(value('overviewKeyFilter')),
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

  function chartEmptyState(message, disconnected) {
    const title = disconnected ? '尚未连接到 CPA' : '暂无可展示的数据';
    const copy = disconnected
      ? '连接 CPA 管理接口后，即可查看 Token、费用和模型调用趋势。'
      : message;
    const action = disconnected ? '<button type="button" class="btn ghost chart-connect-action">连接设置</button>' : '';
    return '<div class="empty-state chart-empty-state">' +
      '<div><div class="chart-empty-icon" aria-hidden="true">'+(disconnected ? '↗' : '—')+'</div>' +
      '<p class="chart-empty-title">'+esc(title)+'</p>' +
      '<p class="chart-empty-copy">'+esc(copy)+'</p>'+action+'</div></div>';
  }

  function renderDisconnectedOverview() {
    ['overviewTrend', 'overviewCostTrend', 'overviewModelShare'].forEach(id => {
      const target = $(id);
      disposeOverviewChart(id);
      target.className = '';
      target.innerHTML = chartEmptyState('', true);
    });
    renderOverviewModelShareLegend([]);
    const rank = $('overviewModelRank');
    if (rank) {
      rank.className = 'model-rank-list';
      rank.innerHTML = chartEmptyState('', true);
    }
    ['overviewTrendTotal', 'overviewCostTrendTotal', 'overviewModelShareTotal', 'overviewRankCount'].forEach(id => {
      const target = $(id);
      if (target) target.textContent = t('未连接');
    });
    const rankHint = $('overviewRankHint');
    if (rankHint) rankHint.textContent = t(modelRankSpec('value').hint);
  }

  function renderDisconnectedTabStates() {
    const content = chartEmptyState('', true).replace('chart-empty-state', 'chart-empty-state tab-empty-state');
    ['keysTable', 'pricingTable', 'usageByKey', 'usageByModel', 'usageRecent'].forEach(id => {
      const target = $(id);
      if (target) target.innerHTML = content;
    });
    const stats = $('usageStats');
    if (stats) stats.innerHTML = content;
    const pagination = $('usagePagination');
    if (pagination) pagination.innerHTML = '';
    [['keysCount', '未连接'], ['modelCatalogCount', '未连接'], ['usageByKeyCount', '未连接'], ['usageByModelCount', '未连接'], ['usageRecentCount', '未连接']].forEach(([id, text]) => {
      const target = $(id);
      if (target) target.textContent = text;
    });
  }

  function renderOverviewEChart(id, emptyText, option, handlers) {
    const target = $(id);
    disposeOverviewChart(id);
    if (!option || !window.echarts) {
      target.className = '';
      target.innerHTML = chartEmptyState(window.echarts ? emptyText : '图表组件加载失败，请检查网络连接后刷新页面', false);
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
    $('overviewModelCount').textContent = models.length ? models.length + ' 个模型' : '—';
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
    if (count) count.textContent = models.length ? models.length + ' 个模型' : '—';
    if (!target) return;
    if (!models.length) {
      target.innerHTML = chartEmptyState(t('当前筛选条件下暂无模型效率数据'), false);
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
    if (filter.plugin_key_id) labels.push('指定 Key');
    if (filter.auth_id || filter.auth_index) labels.push('账号');
    if (filter.model) labels.push('模型: ' + filter.model);
    if (filter.source) labels.push('来源: ' + filter.source);
    $('overviewFilterState').textContent = labels.join(' · ');
    $('overviewStats').innerHTML = [
      ['Key 数 / 可用', keys.length + ' / ' + activeKeys],
      ['筛选请求', totalReq.toLocaleString()],
      ['筛选 Token', formatTokens(totalTokens)],
      ['筛选费用 ' + currencyCode(), formatMoney(totalSpend)],
      ['模型数量', overviewModelUsage(items).length.toLocaleString()],
    ].map(([k,v]) => '<div class="stat"><div class="k">'+esc(k)+'</div><div class="v">'+esc(v)+'</div></div>').join('');
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
      $('keysCount').textContent = state.keys.length ? (state.keys.length + ' 个 Key') : '暂无 Key';
    }

    if (!state.keys.length) {
      $('keysTable').innerHTML = '<div class="keys-empty empty-state"><span>还没有 Key。点击右上角“添加 Key”创建第一个额度凭证。</span></div>';
      return;
    }

    const money = (v) => formatMoney(v);

    const modelChips = (models) => {
      if (!models || !models.length) return '<span class="model-chip all">全部模型</span>';
      const shown = models.slice(0, 3).map(m => '<span class="model-chip" title="'+esc(m)+'">'+esc(m)+'</span>').join('');
      const more = models.length > 3 ? '<span class="model-chip">+'+ (models.length - 3) +'</span>' : '';
      return '<div class="model-chip-row">' + shown + more + '</div>';
    };

    const limitText = (value) => Number(value || 0) > 0 ? money(value) : '不限制';
    const quotaBlock = (k) => {
      const quota = Number(k.quota_micro_usd || 0);
      const used = Number(k.settled_spend_micro_usd || 0);
      const periodLimits = '<span class="quota-periods">' +
        '<span>日 '+esc(limitText(k.daily_quota_micro_usd))+'</span>' +
        '<span>周 '+esc(limitText(k.weekly_quota_micro_usd))+'</span>' +
        '<span>月 '+esc(limitText(k.monthly_quota_micro_usd))+'</span>' +
        '<span>并发 '+esc(Number(k.max_concurrent_requests || 0) || '不限制')+'</span>' +
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

    $('keysTable').innerHTML = '<div class="table-scroll"><table class="keys-table"><thead><tr><th>标签</th><th>可用模型</th><th>Key 限额</th><th>已用 / 剩余</th><th>状态</th><th>操作</th></tr></thead><tbody>' +
      state.keys.map(k => {
        const st = k.revoked_at ? '<span class="badge bad">已删除</span>' : (k.enabled ? '<span class="badge ok">启用</span>' : '<span class="badge warn">禁用</span>');
        return '<tr>' +
          '<td><div class="key-label"><strong title="'+esc(k.label||'(无标签)')+'">'+esc(k.label||'(无标签)')+'</strong></div></td>' +
          '<td>'+modelChips(k.allowed_models)+'</td>' +
          '<td>'+quotaBlock(k)+'</td>' +
          '<td><div class="spend-cell"><span class="primary">'+esc(money(k.settled_spend_micro_usd))+'</span><span class="secondary">剩余 '+(Number(k.quota_micro_usd||0) <= 0 ? '不限制' : esc(money(k.remaining_micro_usd)))+'</span></div></td>' +
          '<td>'+st+'</td>' +
          '<td><div class="row-actions">' +
            '<button class="btn soft sm" data-copy="'+esc(k.id)+'">复制 Key</button>' +
            '<button class="btn primary sm" data-manage="'+esc(k.id)+'">管理 Key</button>' +
            '<button class="btn danger sm" data-delete="'+esc(k.id)+'">删除</button>' +
          '</div></td></tr>';
      }).join('') + '</tbody></table></div>';

    $('keysTable').querySelectorAll('[data-copy]').forEach(btn => btn.addEventListener('click', () => copyKeyByID(btn.dataset.copy)));
    $('keysTable').querySelectorAll('[data-manage]').forEach(btn => btn.addEventListener('click', () => openKeyModal('manage', btn.dataset.manage)));
    $('keysTable').querySelectorAll('[data-delete]').forEach(btn => btn.addEventListener('click', () => openDeleteKeyModal(btn.dataset.delete)));
  }

  function keyFilterLabel(key) {
    return (key.label || '(无标签)') + (key.revoked_at ? '（已删除）' : '') + ' · ' + key.id;
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
    ).join('') : '<div class="key-search-empty">未找到匹配的 Key</div>';
    panel.querySelectorAll('[data-key-id]').forEach(button => button.addEventListener('click', () => {
      const key = state.allKeys.find(item => item.id === button.dataset.keyId);
      if (!key) return;
      input.value = keyFilterLabel(key);
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

  function resolveKeyFilter(raw) {
    raw = String(raw || '').trim();
    if (!raw) return '';
    const exact = state.allKeys.find(key => key.id === raw || keyFilterLabel(key) === raw);
    if (exact) return exact.id;
    const needle = raw.toLocaleLowerCase();
    const matches = state.allKeys.filter(key =>
      key.id.toLocaleLowerCase().includes(needle) || keyFilterLabel(key).toLocaleLowerCase().includes(needle)
    );
    if (matches.length === 1) return matches[0].id;
    if (matches.length > 1) throw new Error('匹配多个 Key，请从搜索建议中选择完整项');
    throw new Error('未找到匹配的 Key');
  }

  function authFilterValue(auth, key) {
    if (!auth) return '';
    const camel = key.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
    for (const name of [key, camel]) if (auth[name] != null) return String(auth[name] || '').trim();
    return '';
  }

  function authFilterLabel(auth) {
    const provider = authFilterValue(auth, 'auth_provider') || authFilterValue(auth, 'provider');
    const account = authFilterValue(auth, 'auth_label') || authFilterValue(auth, 'label')
      || authFilterValue(auth, 'auth_email') || authFilterValue(auth, 'email')
      || authFilterValue(auth, 'auth_name') || authFilterValue(auth, 'name')
      || authFilterValue(auth, 'auth_id') || authFilterValue(auth, 'auth_index') || '未知账号';
    return [provider, account].filter(Boolean).join(' · ');
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
      flash('已复制 Key', true);
    } catch (e) { flash(e.message || '复制失败', false); }
  }

  async function revealKey(id) {
    try {
      const result = await api('POST', 'credit-manager/keys/reveal', { id });
      const key = state.keys.find(item => item.id === id);
      $('keyPlaintext').textContent = result.plaintext || '';
      $('keyPlainResult').classList.remove('hidden');
      $('keyModalHint').textContent = '正在查看「' + (key && key.label || 'Key') + '」的明文凭据；可复制后继续管理。';
      flash('已读取加密保存的 Key 明文', true);
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
      : '未发现可用模型；请确认宿主管理密钥和上游认证文件。';
    refreshCustomControl(picker);
  }

  function selectedKeyModels() {
    return Array.from($('keyModalModels').selectedOptions).map(option => option.value);
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
    const titles = { create: '添加 Key', manage: '管理 Key', rotate: '轮换 Key' };
    const hints = {
      create: '设置额度和模型策略后创建凭据。',
      manage: '可编辑 Key 策略，按需查看明文或轮换凭据。',
      rotate: '创建替代凭据并立即禁用旧 Key；历史使用记录会保留在旧 Key 上。',
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
    $('keyModalEnabled').value = key && !key.enabled ? 'false' : 'true';
    $('keyModalLabel').disabled = isRotation;
    $('keyModalQuotaUSD').disabled = isRotation;
    $('keyModalDailyQuotaUSD').disabled = isRotation;
    $('keyModalWeeklyQuotaUSD').disabled = isRotation;
    $('keyModalMonthlyQuotaUSD').disabled = isRotation;
    $('keyModalMaxConcurrent').disabled = isRotation;
    $('keyModalModels').disabled = isRotation;
    $('keyModalEnabledWrap').classList.toggle('hidden', mode !== 'manage');
    $('keyMaterialWrap').classList.toggle('hidden', mode === 'manage');
    $('keyModalManageActions').classList.toggle('hidden', !isManagedKey);
    $('btnRotateManagedKey').classList.toggle('hidden', Boolean(key && key.revoked_at));
    $('keyMaterial').value = '';
    $('keyPlainResult').classList.add('hidden');
    $('btnSubmitKeyModal').textContent = mode === 'create' ? '创建 Key' : (mode === 'manage' ? '保存策略' : '确认轮换');
    refreshCustomControls();
    const modal = $('keyModal');
    modal.classList.add('open');
    modal.setAttribute('aria-hidden', 'false');
    $('keyModalLabel').focus();
    void loadKeyModels(allowedModels);
  }

  function closeKeyModal() {
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
    $('priceEnabled').value = ruleEnabled(rule) ? 'true' : 'false';
    $('priceModelPicker').value = '';
    $('modelPriceStatus').textContent = rule ? '正在编辑「' + id + '」。可同步并选择当前可用模型的 models.dev 价格进行回填。' : '价格来源：models.dev。仅匹配当前代理公开的模型；同步不会自动保存。';
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
        allowed_models: selectedKeyModels(),
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
        enabled: $('keyModalEnabled').value === 'true',
        set_allowed_models: true,
        allowed_models: selectedKeyModels(),
      });
    } else {
      result = await api('POST', 'credit-manager/keys/rotate', { id, key_material: keyMaterial });
    }
    await reload();
    if (mode === 'manage') {
      closeKeyModal();
      flash('Key 策略已保存', true);
      return;
    }
    if (mode === 'create' && result.plaintext) {
      try {
        await copyText(result.plaintext);
        closeKeyModal();
        flash('Key 已创建并复制到剪贴板', true);
      } catch (_) {
        $('keyPlaintext').textContent = result.plaintext;
        $('keyPlainResult').classList.remove('hidden');
        flash('Key 已创建；自动复制失败，请手动复制', false);
      }
      return;
    }
    if (result.plaintext) {
      $('keyPlaintext').textContent = result.plaintext;
      $('keyPlainResult').classList.remove('hidden');
    }
    flash(mode === 'rotate' ? 'Key 已轮换，旧 Key 已禁用；请复制新 Key' : 'Key 已创建；请复制明文', true);
  }

  function priceNumber(value) {
    const number = Number(value);
    return Number.isFinite(number) && number >= 0 ? number : 0;
  }

  function modelIDs(payload) {
    const items = Array.isArray(payload) ? payload : (payload && payload.data) || [];
    return [...new Set(items.map(item => typeof item === 'string' ? item : item && item.id).filter(Boolean))].sort();
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
    return [c.input, c.output, c.cache_read, c.cache_write].map(value => Number(value) || 0).join('|');
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

  function modelsDevPricingRule(modelID, item) {
    const cost = item.cost || {};
    return {
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
    };
  }

  async function syncModelsDevPrices(prices, rules) {
    const exactRules = pricingRuleIndex(rules);
    const ruleIDs = new Set((rules || []).map(rule => String(rule.ID || rule.id || '').trim()));
    const pending = Object.entries(prices).filter(([modelID, item]) => item && item.tokenPriced && !exactRules.has(modelID) && !ruleIDs.has(modelID));
    let saved = 0;
    const failed = [];
    for (const [modelID, item] of pending) {
      try {
        await api('POST', 'credit-manager/pricing', modelsDevPricingRule(modelID, item));
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
    const status = $('modelCatalogStatus');
    const button = $('btnLoadModelCatalog');
    button.disabled = true;
    status.textContent = '正在读取当前代理模型与价格目录…';
    try {
      const catalogResult = await Promise.allSettled([
        fetchAvailableModels(),
        fetchModelsDevCatalog(false),
      ]);
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
        ].map(([label, value]) => '<span class="price-pair"><span>'+esc(label)+'</span><strong>'+esc(priceNumber(value))+'</strong></span>').join('') : (imageGen ? '<span class="muted">按张计费，不能套用 Token 价</span>' : '<span class="muted">未找到唯一匹配价格</span>');
        const ruleBilling = priceValue(price, 'BillingMode', 'billing_mode');
        const currentRule = rule ? '<div class="pricing-rule-id mono">'+esc(rule.ID || rule.id)+'</div><div class="hint">'+(ruleBilling === 'per_image' ? '每张 '+esc(formatMoney(priceValue(price, 'PerImage', 'per_image'))) : '输入 '+esc(formatMoney(priceValue(price, 'Input', 'input')))+' · 输出 '+esc(formatMoney(priceValue(price, 'Output', 'output'))))+'</div>' : '<span class="muted">未设置</span>';
        const status = rule ? (enabled ? '<span class="badge ok">启用</span>' : '<span class="badge warn">禁用</span>') : (tokenPriced ? '<span class="badge">待保存</span>' : (imageGen ? '<span class="badge warn">出图</span>' : '<span class="badge bad">无价格</span>'));
        return '<tr'+(!enabled ? ' class="is-disabled"' : '')+'>' +
          '<td><div class="pricing-rule-id mono">'+esc(modelID)+'</div>'+(matched && matched.provider ? '<div class="hint">models.dev / '+esc(matched.provider)+(imageGen ? ' · 出图' : '')+'</div>' : (imageGen ? '<div class="hint">出图模型</div>' : ''))+'</td>' +
          '<td><div class="price-pairs">'+pricePairs+'</div></td>' +
          '<td>'+currentRule+'</td>' +
          '<td><div class="pricing-meta">'+status+'</div></td>' +
          '<td><div class="actions pricing-actions"><button class="btn '+(enabled ? 'ghost' : 'soft')+' sm" data-toggle-price="'+esc(modelID)+'">'+(enabled ? '禁用' : '启用')+'</button><button class="btn soft sm" data-configure-model="'+esc(modelID)+'">'+(rule ? '编辑' : '设置价格')+'</button>'+(rule ? '<button class="btn danger sm" data-del-price="'+esc(rule.ID || rule.id)+'">删除</button>' : '')+'</div></td>' +
          '</tr>';
      }).join('') + '</tbody></table></div>';
    $('pricingTable').querySelectorAll('[data-toggle-price]').forEach(btn => btn.addEventListener('click', async () => {
      const modelID = btn.dataset.togglePrice;
      const rule = rulesByModel.get(modelID);
      const nextEnabled = !modelIsDisabled(modelID, rules);
      btn.disabled = true;
      try {
        await setModelPricingEnabled(modelID, nextEnabled);
      } catch (e) { flash(e.message, false); }
      finally { btn.disabled = false; }
    }));
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
      label: x.Label || x.label || '历史 Key',
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
      ['筛选请求数', totalRequests],
      ['筛选费用 ' + currencyCode(), formatMoney(totalCost)],
      ['输入 Token', formatTokens(totalInput)],
      ['输出 Token', formatTokens(totalOutput)],
    ].map(([k,v]) => '<div class="stat"><div class="k">'+esc(k)+'</div><div class="v">'+esc(v)+'</div></div>').join('');
    $('usageByKeyCount').textContent = byKey.length + ' 个 Key';
    $('usageByModelCount').textContent = byModel.length + ' 个模型';
    $('usageRecentCount').textContent = (Number(pageData.total || 0)).toLocaleString() + ' 条明细';
    const filterLabels = [];
    const filter = getUsageFilterValues();
    const rangeText = $('usageRangeFilter').selectedOptions[0].textContent;
    if (rangeText) filterLabels.push(rangeText);
    if (filter.plugin_key_id) filterLabels.push('Key');
    if (filter.auth_id || filter.auth_index) filterLabels.push('账号');
    if (filter.model) filterLabels.push('模型: ' + filter.model);
    if (filter.source) filterLabels.push('来源: ' + filter.source);
    if (filter.min_cost_micro_usd || filter.max_cost_micro_usd) filterLabels.push('费用范围');
    if (filter.min_tokens || filter.max_tokens) filterLabels.push('Token 范围');
    $('usageFilterState').textContent = filterLabels.length ? filterLabels.join(' · ') : '未设置筛选';

    const emptyState = message => '<div class="empty-state"><span>'+esc(message)+'</span></div>';
    $('usageByKey').innerHTML = byKey.length ? '<div class="table-scroll"><table><thead><tr><th>Key</th><th>请求数</th><th>费用 '+esc(currencyCode())+'</th><th>in/out tokens</th></tr></thead><tbody>' +
      byKey.map(x => '<tr><td><div><strong>'+esc(x.label||'(无标签)')+'</strong></div></td><td>'+esc(x.req)+'</td><td title="'+esc(moneyTitle(x.cost))+'">'+esc(formatMoney(x.cost))+'</td><td title="'+esc(tokenTitle(x.inn)+' / '+tokenTitle(x.out))+'">'+esc(formatTokens(x.inn))+' / '+esc(formatTokens(x.out))+'</td></tr>').join('') +
      '</tbody></table></div><p class="table-swipe-hint">左右滑动查看完整汇总</p>' : emptyState('当前筛选条件下暂无按 Key 汇总');
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
      const display = [provider, account].filter(Boolean).join(' · ') || '未知账号';
      return '<td class="usage-key" title="'+esc(display)+'"><div class="usage-key-label"><strong>'+esc(display)+'</strong></div></td>';
    };

    $('usageRecent').innerHTML = items.length ? '<div class="table-scroll"><table class="usage-table"><thead><tr><th>时间</th><th>账号</th><th>模型名称</th><th>来源</th><th>结果</th><th>首字延迟</th><th>生成时间</th><th>TPS</th><th>思考强度</th><th>输入</th><th>输出</th><th>思考</th><th>缓存读取</th><th>缓存创建</th><th>总 Token 数</th><th>缓存命中</th><th>费用 '+esc(currencyCode())+'</th></tr></thead><tbody>' +
      items.map(u => {
        const settledCost = u.cost_micro_usd;
        const cell = (value) => '<td title="'+esc(tokenTitle(value))+'">'+esc(formatTokens(value))+'</td>';
        return '<tr><td class="mono" title="'+esc(u.created_at)+'">'+esc(formatDateTime(u.created_at))+'</td>'+usageAuthCell(u)+
          '<td class="mono model" title="'+esc(u.model)+'">'+esc(u.model)+'</td><td>'+esc(u.source || '—')+'</td><td>'+esc(u.result || '—')+'</td><td>'+esc(formatMilliseconds(u.first_token_latency_ms))+'</td><td>'+esc(formatMilliseconds(u.generation_duration_ms))+'</td><td>'+esc(formatTPS(u.tokens_per_second))+'</td><td>'+esc(u.thinking_intensity || '—')+'</td>'+
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
    $('usagePagination').innerHTML = '<span class="muted">显示 '+start+'–'+end+'，共 '+total.toLocaleString()+' 条</span>' +
      '<label>每页<select id="usagePageSize"><option value="25">25 条</option><option value="50">50 条</option><option value="100">100 条</option><option value="200">200 条</option></select></label>' +
      '<button class="btn ghost" id="btnUsagePrev" '+(page <= 1 ? 'disabled' : '')+'>上一页</button>' +
      '<span class="muted">第 '+page+' / '+totalPages+' 页</span>' +
      '<button class="btn ghost" id="btnUsageNext" '+(page >= totalPages ? 'disabled' : '')+'>下一页</button>';
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
      plugin_key_id: resolveKeyFilter(value('usageKeyFilter')),
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
    const [summary, recent] = await Promise.all([
      api('GET', 'credit-manager/usage/summary' + usageQuery(false)),
      api('GET', 'credit-manager/usage' + usageQuery(true)),
    ]);
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
    return (current ? '当前 · ' : '') + authQuotaShortTime(authQuotaValue(window, 'cycle_start_at')) + ' → ' + authQuotaShortTime(authQuotaValue(window, 'resets_at'));
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



  function authQuotaProviderName(provider) {
    const value = String(provider || '').trim();
    if (!value) return '未知提供商';
    const key = value.toLowerCase();
    if (key === 'xai' || key === 'grok') return 'xAI';
    if (key === 'codex' || key === 'openai') return 'Codex';
    if (key === 'claude' || key === 'anthropic') return 'Claude';
    if (key === 'antigravity' || key === 'google') return 'Antigravity';
    if (key === 'kimi' || key === 'moonshot') return 'Kimi';
    return value;
  }

  function syncAuthQuotaProviderFilter(items) {
    const select = $('authQuotaProviderFilter');
    if (!select) return;
    const providers = Array.from(new Set(items.map(item => String(authQuotaValue(item, 'provider') || '').trim()).filter(Boolean))).sort((a, b) => authQuotaProviderName(a).localeCompare(authQuotaProviderName(b), 'zh'));
    if (!providers.includes(state.authQuotaProvider)) state.authQuotaProvider = '';
    select.innerHTML = '<option value="">全部平台</option>' + providers.map(provider => '<option value="'+esc(provider)+'">'+esc(authQuotaProviderName(provider))+'</option>').join('');
    select.value = state.authQuotaProvider;
  }

  function authQuotaMatchesFilters(item) {
    const provider = String(authQuotaValue(item, 'provider') || '');
    if (state.authQuotaProvider && provider !== state.authQuotaProvider) return false;
    const query = String(state.authQuotaName || '').trim().toLowerCase();
    if (!query) return true;
    const haystack = [authQuotaValue(item, 'display_name'), authQuotaValue(item, 'auth_id'), authQuotaValue(item, 'auth_index'), provider, authQuotaProviderName(provider)].map(value => String(value || '').toLowerCase()).join(' ');
    return haystack.includes(query);
  }

  function authQuotaBadge(status) {
    switch (String(status || '').toLowerCase()) {
      case 'fresh': return { tone: 'ok', text: '最新' };
      case 'stale': return { tone: 'warn', text: '缓存过期' };
      case 'unavailable': return { tone: 'bad', text: '不可用' };
      default: return { tone: 'warn', text: authQuotaText(status) };
    }
  }

  function renderAuthQuotas(result) {
    if (result) state.authQuotas = result;
    const items = state.authQuotas && Array.isArray(authQuotaValue(state.authQuotas, 'items')) ? authQuotaValue(state.authQuotas, 'items') : [];
    if (!items.length) {
      state.authQuotaWeeks = {};
      syncAuthQuotaProviderFilter([]);
      $('authQuotaList').innerHTML = emptyState('当前没有可用的认证额度数据');
      return;
    }
    syncAuthQuotaProviderFilter(items);
    const visibleItems = items.filter(authQuotaMatchesFilters);
    if (!visibleItems.length) {
      $('authQuotaList').innerHTML = emptyState('没有符合筛选条件的认证额度');
      return;
    }
    const now = Date.now();
    $('authQuotaList').innerHTML = visibleItems.map(item => {
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
        const period = authQuotaPeriodLabel(window);
        const used = authQuotaAmount(window, 'used');
        const remaining = authQuotaAmount(window, 'remaining');
        const progress = ratio.known ? ratio.percent+'%' : '—';
        const progressClass = ratio.known ? ratio.tone : 'unknown';
        return '<section class="auth-quota-window-card">'+
          '<div class="auth-quota-progress '+progressClass+'" style="--quota-progress:'+ratio.percent+'%" role="img" aria-label="'+esc(label)+' 已用 '+progress+'"><span>'+esc(progress)+'</span></div>'+
          '<div class="auth-quota-window-content"><div class="auth-quota-window-name" title="'+esc(label)+'">'+esc(label)+(period ? '<span class="auth-quota-period">'+esc(period)+'</span>' : '')+'</div><div class="auth-quota-window-stats"><span>已用<strong>'+esc(used)+'</strong></span><span>剩余<strong>'+esc(remaining)+'</strong></span></div><div class="auth-quota-window-reset" title="重置时间 '+esc(authQuotaTime(authQuotaValue(window, 'resets_at')))+'">重置 '+esc(authQuotaShortTime(authQuotaValue(window, 'resets_at')))+'</div></div>'+
          '</section>';
      }).join('') : '<div class="empty-state">该额度周暂无窗口</div>';
      const weeklyForCost = (Array.isArray(windows) ? windows : []).filter(window => authQuotaIsWeekly(window) && authQuotaWeekKey(window) === selected);
      const costs = authQuotaCostForecast(weeklyForCost.length ? weeklyForCost : visible);
      const weekSelect = weeks.length
        ? '<label class="auth-quota-filter"><span>额度周</span><select class="auth-quota-week-select" data-auth-id="'+esc(itemKey)+'" title="额度周">'+weeks.map(week => '<option value="'+esc(week.key)+'"'+(week.key === selected ? ' selected' : '')+'>'+esc(week.label)+'</option>').join('')+'</select></label>'
        : '<label class="auth-quota-filter"><span>额度周</span><select disabled title="额度周"><option>暂无额度周</option></select></label>';
      return '<article class="card auth-quota-card"><header class="auth-quota-header"><div class="auth-quota-identity"><div class="auth-quota-identity-row"><p class="auth-quota-provider">'+esc(authQuotaValue(item, 'provider') || '未知提供商')+'</p><span class="badge '+badge.tone+'">'+esc(badge.text)+'</span></div><h2 class="auth-quota-title" title="'+esc(authQuotaValue(item, 'display_name') || '未命名认证')+'">'+esc(authQuotaValue(item, 'display_name') || '未命名认证')+'</h2><p class="auth-quota-sync">同步 '+esc(authQuotaShortTime(authQuotaValue(item, 'last_success_at')))+'</p></div><div class="auth-quota-cost-grid"><div class="auth-quota-cost"><span>当前费用</span><strong title="'+esc(costs.used)+'">'+esc(costs.used)+'</strong></div><div class="auth-quota-cost"><span>预估剩余</span><strong title="'+esc(costs.remaining)+'">'+esc(costs.remaining)+'</strong></div><div class="auth-quota-cost"><span>预计可用</span><strong title="'+esc(costs.available)+'">'+esc(costs.available)+'</strong></div></div><div class="auth-quota-header-tools">'+weekSelect+'</div></header>'+(error ? '<div class="auth-quota-error">'+esc(error)+'</div>' : '')+'<div class="auth-quota-window-grid">'+cards+'</div></article>';
    }).join('');
  }

  async function loadAuthQuotas() {
    renderAuthQuotas(await api('GET', 'credit-manager/auth-quotas'));
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
      flash('Key 已删除，历史使用统计已保留', true);
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
    $('apiBase').focus();
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
    }
  });
  $('btnSaveToken').addEventListener('click', async () => {
    const t = $('mgmtToken').value.trim();
    if (!t) { flash('请输入管理密钥', false); return; }
    const base = $('apiBase').value.trim().replace(/\/$/, '');
    sessionStorage.setItem(TOKEN_KEY, t);
    if (base) sessionStorage.setItem(BASE_KEY, base);
    else sessionStorage.removeItem(BASE_KEY);
    try {
      await reloadWithModelCatalog();
      closeConnectionModal();
      flash('已加载数据', true);
    } catch (e) { flash(e.message, false); }
  });
  $('btnClearToken').addEventListener('click', () => {
    sessionStorage.removeItem(TOKEN_KEY);
    sessionStorage.removeItem(BASE_KEY);
    $('mgmtToken').value = '';
    flash('已清除本地管理密钥与 API 根地址', true);
  });
  $('authQuotaProviderFilter').addEventListener('change', () => {
    state.authQuotaProvider = $('authQuotaProviderFilter').value || '';
    renderAuthQuotas();
  });
  $('authQuotaNameFilter').addEventListener('input', () => {
    state.authQuotaName = $('authQuotaNameFilter').value || '';
    renderAuthQuotas();
  });
  $('authQuotaList').addEventListener('change', event => {
    const select = event.target.closest('.auth-quota-week-select');
    if (!select) return;
    const authID = select.getAttribute('data-auth-id') || '';
    if (!authID) return;
    state.authQuotaWeeks[authID] = select.value || '';
    renderAuthQuotas();
  });
  $('btnLoadAuthQuotas').addEventListener('click', async () => {
    try { await loadAuthQuotas(); flash('认证额度已刷新', true); }
    catch (e) { flash(e.message, false); }
  });

  $('btnRefresh').addEventListener('click', async () => {
    try {
      if (state.currentTab === 'auth-quotas') { await loadAuthQuotas(); flash('认证额度已重新加载', true); }
      else { await reloadWithModelCatalog(); flash('数据已刷新', true); }
    } catch (e) { flash(e.message, false); }
  });
  $('overviewRangeFilter').addEventListener('change', setOverviewRangeVisibility);
  $('usageRangeFilter').addEventListener('change', setUsageRangeVisibility);
  ['overview', 'usage'].forEach(kind => {
    const input = $(kind + 'KeyFilter');
    input.addEventListener('focus', () => openKeySearch(kind));
    input.addEventListener('input', () => {
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
      flash('已复制 Key', true);
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
  $('btnSavePrice').addEventListener('click', async () => {
    try {
      const body = {
        id: $('priceId').value.trim(),
        match_kind: $('priceKind').value,
        pattern: $('pricePattern').value.trim(),
        priority: Number($('pricePriority').value || 0),
        enabled: $('priceEnabled').value === 'true',
        price: {
          input: microFromUSD($('priceIn').value) || 0,
          output: microFromUSD($('priceOut').value) || 0,
          cache_read: microFromUSD($('priceCacheRead').value) || 0,
          cache_creation: microFromUSD($('priceCacheCreation').value) || 0,
          accounting_mode: $('priceAccountingMode').value || '',
          billing_mode: $('priceBillingMode').value || 'token',
          per_image: microFromUSD($('pricePerImage').value) || 0,
        },
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
  const saved = sessionStorage.getItem(TOKEN_KEY) || '';
  if (saved) {
    $('mgmtToken').value = saved;
    reloadWithModelCatalog().catch(e => flash(e.message, false));
  } else {
    renderDisconnectedOverview();
    renderDisconnectedTabStates();
  }
})();
