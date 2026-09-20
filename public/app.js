/* ================= Anime-Wor — основной клиент ================= */

let DATA = { anime: [], banners: [] };
let CURRENT_USER = null;
let USER_FAVORITES = [];

// ---------- ТЕМА ----------
function toggleTheme() {
  const html = document.documentElement;
  const isLight = html.classList.toggle('theme-light');
  localStorage.setItem('aw_theme', isLight ? 'light' : 'dark');
  const btn = document.getElementById('themeToggle');
  if (btn) btn.textContent = isLight ? '☀️' : '🌙';
}

function initTheme() {
  const theme = localStorage.getItem('aw_theme') || 'dark';
  const btn = document.getElementById('themeToggle');
  if (btn) btn.textContent = theme === 'light' ? '☀️' : '🌙';
}

// ---------- API helper ----------
async function api(path, opts = {}) {
  const res = await fetch('/api' + path, {
    method: opts.method || 'GET',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'same-origin',
    body: opts.body ? JSON.stringify(opts.body) : undefined
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || 'Ошибка запроса');
  return data;
}

// ---------- УТИЛИТЫ ----------
function toast(msg, isError = false) {
  const t = document.getElementById('toastBox');
  if (!t) return;
  t.textContent = msg;
  t.classList.toggle('error', isError);
  t.classList.add('show');
  clearTimeout(t._timer);
  t._timer = setTimeout(() => t.classList.remove('show'), 2800);
}

function escapeHtml(s) {
  return String(s || '').replace(/[&<>"']/g, c => ({
    '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'
  }[c]));
}

// ---------- ЗАГРУЗКА ДАННЫХ ----------
async function loadAll() {
  try {
    const [a, b] = await Promise.all([
      api('/anime'),
      api('/banners')
    ]);
    DATA.anime = a.anime || [];
    DATA.banners = b.banners || [];
  } catch (e) {
    toast('Не удалось загрузить данные: ' + e.message, true);
  }
}

function getAnime(id) {
  return DATA.anime.find(a => a.id === id);
}

// ---------- НАВИГАЦИЯ ----------

async function renderSchedule() {
  try {
    const { schedule } = await api('/schedule');
    const weekdays = ['', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
    const fullNames = ['', 'Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота', 'Воскресенье'];

    const byDay = { 1: [], 2: [], 3: [], 4: [], 5: [], 6: [], 7: [] };
    schedule.forEach(item => {
      if (byDay[item.weekday]) byDay[item.weekday].push(item);
    });

    const daysHTML = [1,2,3,4,5,6,7].map(d => `
      <div class="schedule-day">
        <h4>${fullNames[d]}</h4>
        ${byDay[d].length
          ? byDay[d].map(it => `
              <div class="schedule-item" onclick="navigate('anime','${it.anime_id}')">
                <img src="${escapeHtml(it.poster || '')}" onerror="this.src='https://via.placeholder.com/34x46/FF6B35/fff'">
                <div class="schedule-item-info">
                  <b>${escapeHtml(it.title || 'Без названия')}</b>
                  <span>${it.episode ? 'Серия ' + it.episode : ''} ${it.air_time || ''}</span>
                </div>
              </div>`).join('')
          : '<p style="font-size:12px; color:var(--text-3)">Пусто</p>'}
      </div>`).join('');

    return `<div class="container">
      <section class="section">
        <div class="section-head"><h2>Расписание онгоингов</h2></div>
        <div class="schedule-grid">${daysHTML}</div>
      </section>
    </div>`;
  } catch (e) {
    return `<div class="container"><div class="empty"><h3>Ошибка загрузки расписания</h3><p>${escapeHtml(e.message)}</p></div></div>`;
  }
}

async function navigate(page, param) {
  window.scrollTo({ top: 0 });
  const app = document.getElementById('app');
  if (!app) return;

  if (page === 'home')            app.innerHTML = renderHome();
  else if (page === 'catalog')    app.innerHTML = renderCatalog();
  else if (page === 'schedule')   app.innerHTML = await renderSchedule();
  else if (page === 'favorites')  app.innerHTML = await renderFavorites();
  else if (page === 'anime') {
  app.innerHTML = renderAnimePage(param);
  loadRating(param);
  loadComments(param);
  }
  else if (page === 'admin') {
    if (!CURRENT_USER || CURRENT_USER.role !== 'admin') {
      app.innerHTML = `<div class="container"><div class="empty"><h3>Доступ запрещён</h3><p>Только для админа</p></div></div>`;
      return;
    }
    app.innerHTML = renderAdmin();
    if (typeof initAdmin === 'function') await initAdmin();
  } else {
    app.innerHTML = renderHome();
  }

  if (page === 'home') initHeroSlider();
}

// ---------- ГЛАВНАЯ ----------
function renderHome() {
  const banners = DATA.banners;
  const newAnime = [...DATA.anime].slice(0, 8);
  const popular = [...DATA.anime].sort((a, b) => b.rating - a.rating).slice(0, 8);

  const slides = banners.map((b, i) => {
    const a = getAnime(b.anime_id);
    return `<div class="hero-slide ${i === 0 ? 'active' : ''}" style="background-image:url('${escapeHtml(b.image)}')">
      <div class="hero-content">
        <span class="hero-badge">${a ? escapeHtml(a.status) : 'Премьера'}</span>
        <h2>${escapeHtml(b.title)}</h2>
        <p>${escapeHtml(b.subtitle || '')}</p>
        <button class="btn" onclick="navigate('anime','${b.anime_id}')">▶ Смотреть</button>
      </div>
    </div>`;
  }).join('');

  const dots = banners.map((_, i) =>
    `<span class="${i === 0 ? 'active' : ''}" data-idx="${i}"></span>`
  ).join('');

  return `<div class="container">
    ${banners.length ? `<div class="hero" id="hero">${slides}<div class="hero-dots" id="heroDots">${dots}</div></div>` : ''}

    <section class="section">
      <div class="section-head">
        <h2>Новинки</h2>
        <button class="btn btn-ghost btn-sm" onclick="navigate('catalog')">Все →</button>
      </div>
      <div class="grid">${newAnime.map(cardHTML).join('') || '<div class="empty"><h3>Пусто</h3></div>'}</div>
    </section>

    <section class="section">
      <div class="section-head">
        <h2>Популярное</h2>
        <button class="btn btn-ghost btn-sm" onclick="navigate('catalog')">Все →</button>
      </div>
      <div class="grid">${popular.map(cardHTML).join('')}</div>
    </section>
  </div>`;
}

function cardHTML(a) {
  return `<div class="card" onclick="navigate('anime','${a.id}')">
    <div class="card-poster">
      <img src="${escapeHtml(a.poster)}" alt="${escapeHtml(a.title)}"
           onerror="this.src='https://via.placeholder.com/300x450/FFDE00/000000?text=AW'">
      <span class="card-rating">★ ${a.rating}</span>
      <span class="card-status">${escapeHtml(a.status)}</span>
    </div>
    <div class="card-body">
      <h3>${escapeHtml(a.title)}</h3>
      <div class="meta">
        <span>${a.year}</span>·<span>${escapeHtml(a.genres[0] || '')}</span>
      </div>
    </div>
  </div>`;
}

let catalogFilters = { q: '', genre: '', year: '', status: '', sort: 'new' };

function renderCatalog() {
  // Собираем все жанры и годы
  const genres = new Set();
  const years = new Set();
  DATA.anime.forEach(a => {
    a.genres.forEach(g => genres.add(g));
    years.add(a.year);
  });
  const genreList = [...genres].sort();
  const yearList = [...years].sort((a, b) => b - a);

  let list = DATA.anime.slice();

  // Фильтрация
  if (catalogFilters.q) {
    const q = catalogFilters.q.toLowerCase();
    list = list.filter(a =>
      a.title.toLowerCase().includes(q) ||
      (a.original || '').toLowerCase().includes(q)
    );
  }
  if (catalogFilters.genre) list = list.filter(a => a.genres.includes(catalogFilters.genre));
  if (catalogFilters.year)  list = list.filter(a => String(a.year) === catalogFilters.year);
  if (catalogFilters.status) list = list.filter(a => a.status === catalogFilters.status);

  // Сортировка
  if (catalogFilters.sort === 'rating') list.sort((a, b) => b.rating - a.rating);
  else if (catalogFilters.sort === 'year') list.sort((a, b) => b.year - a.year);
  else if (catalogFilters.sort === 'title') list.sort((a, b) => a.title.localeCompare(b.title));

  return `<div class="container">
    <section class="section">
      <div class="section-head"><h2>Каталог</h2></div>

      <div class="filters">
        <input id="f_q" placeholder="Поиск..." value="${escapeHtml(catalogFilters.q)}" oninput="applyFilter('q', this.value)">
        <select onchange="applyFilter('genre', this.value)">
          <option value="">Все жанры</option>
          ${genreList.map(g => `<option value="${escapeHtml(g)}" ${catalogFilters.genre === g ? 'selected' : ''}>${escapeHtml(g)}</option>`).join('')}
        </select>
        <select onchange="applyFilter('year', this.value)">
          <option value="">Все годы</option>
          ${yearList.map(y => `<option value="${y}" ${catalogFilters.year === String(y) ? 'selected' : ''}>${y}</option>`).join('')}
        </select>
        <select onchange="applyFilter('status', this.value)">
          <option value="">Все статусы</option>
          <option value="Онгоинг" ${catalogFilters.status === 'Онгоинг' ? 'selected' : ''}>Онгоинг</option>
          <option value="Завершён" ${catalogFilters.status === 'Завершён' ? 'selected' : ''}>Завершён</option>
          <option value="Анонс" ${catalogFilters.status === 'Анонс' ? 'selected' : ''}>Анонс</option>
        </select>
        <select onchange="applyFilter('sort', this.value)">
          <option value="new" ${catalogFilters.sort === 'new' ? 'selected' : ''}>Сначала новые</option>
          <option value="rating" ${catalogFilters.sort === 'rating' ? 'selected' : ''}>По рейтингу</option>
          <option value="year" ${catalogFilters.sort === 'year' ? 'selected' : ''}>По году</option>
          <option value="title" ${catalogFilters.sort === 'title' ? 'selected' : ''}>По алфавиту</option>
        </select>
        <button class="chip" onclick="resetFilters()">Сбросить</button>
      </div>

      <div class="grid">
        ${list.map(cardHTML).join('') || '<div class="empty"><h3>Ничего не найдено</h3></div>'}
      </div>
    </section>
  </div>`;
}

function applyFilter(key, value) {
  catalogFilters[key] = value;
  const app = document.getElementById('app');
  app.innerHTML = renderCatalog();
  // Возвращаем фокус в поле поиска
  if (key === 'q') {
    const inp = document.getElementById('f_q');
    if (inp) { inp.focus(); inp.setSelectionRange(inp.value.length, inp.value.length); }
  }
}

function resetFilters() {
  catalogFilters = { q: '', genre: '', year: '', status: '', sort: 'new' };
  document.getElementById('app').innerHTML = renderCatalog();
}

async function renderFavorites() {
  if (!CURRENT_USER) {
    return `<div class="container">
      <div class="empty">
        <h3>Войдите в аккаунт</h3>
        <p>Избранное доступно авторизованным пользователям</p>
        <button class="btn" style="margin-top:16px" onclick="openLogin()">Войти</button>
      </div>
    </div>`;
  }
  try {
    const { favorites } = await api('/users/favorites');
    USER_FAVORITES = favorites;
  } catch {}
  const list = DATA.anime.filter(a => USER_FAVORITES.includes(a.id));
  return `<div class="container">
    <section class="section">
      <div class="section-head"><h2>Избранное</h2></div>
      <div class="grid">
        ${list.map(cardHTML).join('') || '<div class="empty"><h3>Список пуст</h3><p>Добавляйте аниме в избранное ♡</p></div>'}
      </div>
    </section>
  </div>`;
}

// ---------- СТРАНИЦА АНИМЕ ----------
function renderAnimePage(id) {
  const a = getAnime(id);
  if (!a) return `<div class="container"><div class="empty"><h3>Аниме не найдено</h3></div></div>`;

  const isFav = USER_FAVORITES.includes(a.id);
  const voicesHTML = a.voices.map((v, vi) =>
    `<button class="voice-btn ${vi === 0 ? 'active' : ''}" onclick="selectVoice('${a.id}', ${vi})">${escapeHtml(v.name)}</button>`
  ).join('');

  const eps = a.voices[0]?.episodes || [];
  const epsHTML = eps.map((_, i) =>
    `<button class="ep-btn ${i === 0 ? 'active' : ''}" onclick="selectEpisode('${a.id}', ${i}, this)">${i + 1}</button>`
  ).join('');

  return `<div class="container">
    <div class="anime-page">
      <div class="anime-poster">
        <img src="${escapeHtml(a.poster)}" alt="${escapeHtml(a.title)}"
             onerror="this.src='https://via.placeholder.com/300x450/FF6B35/fff?text=AW'">
        <button class="btn btn-block" onclick="document.getElementById('player').scrollIntoView({behavior:'smooth'})">▶ Смотреть</button>
        ${CURRENT_USER ? `<button class="btn btn-ghost btn-block" style="margin-top:8px" onclick="toggleFavorite('${a.id}')">${isFav ? '★ В избранном' : '☆ В избранное'}</button>` : ''}
      </div>

      <div class="anime-info">
        <h1>${escapeHtml(a.title)}</h1>
        <p class="sub">${escapeHtml(a.original)} · ${a.year}</p>

        <div class="tags">${a.genres.map(g => `<span class="tag">${escapeHtml(g)}</span>`).join('')}</div>

        <div class="info-row">
          <div class="info-item"><b>Рейтинг</b><span id="ratingVal">★ ${a.rating}</span></div>
          <div class="info-item"><b>Статус</b><span>${escapeHtml(a.status)}</span></div>
          <div class="info-item"><b>Эпизодов</b><span>${eps.length}</span></div>
          <div class="info-item"><b>Год</b><span>${a.year}</span></div>
        </div>

        <div class="rating-block" id="userRating">
          <div>
            <div class="rating-big" id="avgRating">—</div>
            <div class="rating-info" id="ratingCount">нет оценок</div>
          </div>
          <div style="flex:1"></div>
          ${CURRENT_USER
            ? `<div class="rating-stars" id="ratingStars">
                ${[1,2,3,4,5,6,7,8,9,10].map(n => `<button onclick="setRating('${a.id}', ${n})" data-score="${n}">${n}</button>`).join('')}
              </div>`
            : `<span style="font-size:12.5px;color:var(--text-3)"><a href="#" onclick="openLogin();return false;" style="color:var(--accent)">Войдите</a>, чтобы оценить</span>`}
        </div>

        <p class="anime-description">${escapeHtml(a.description)}</p>

        <div class="voices">
          <h3>Выбор озвучки</h3>
          <div class="voice-list">${voicesHTML || '<p style="color:var(--text-3);font-size:13px">Озвучки пока нет</p>'}</div>
        </div>

        <div class="episodes">
          <h3>Эпизоды</h3>
          <div class="episode-list" id="epList">${epsHTML || '<p style="color:var(--text-3);font-size:13px">Эпизоды не загружены</p>'}</div>
        </div>

        <div class="player" id="player">
          ${eps[0]
            ? `<iframe id="playerFrame" src="${escapeHtml(eps[0])}" allowfullscreen allow="autoplay; encrypted-media"></iframe>`
            : `<div style="color:#fff;display:grid;place-items:center;height:100%;font-size:14px">Видео не добавлено</div>`}
        </div>

        <div class="comments">
          <h3>Комментарии <span id="commentsCount" style="color:var(--text-3);font-weight:500"></span></h3>
          ${CURRENT_USER
            ? `<div class="comment-form">
                <textarea id="commentText" placeholder="Написать комментарий..."></textarea>
                <button class="btn" onclick="sendComment('${a.id}')">Отправить</button>
              </div>`
            : `<p style="color:var(--text-3);font-size:13.5px;margin-bottom:18px"><a href="#" onclick="openLogin();return false;" style="color:var(--accent)">Войдите</a>, чтобы оставить комментарий</p>`}
          <div id="commentsList"><p style="color:var(--text-3);font-size:13px">Загрузка...</p></div>
        </div>
      </div>
    </div>
  </div>`;
}

function selectVoice(animeId, vi) {
  const a = getAnime(animeId);
  if (!a) return;
  document.querySelectorAll('.voice-btn').forEach((b, i) => b.classList.toggle('active', i === vi));
  const eps = a.voices[vi].episodes;
  const epsHTML = eps.map((_, i) =>
    `<button class="ep-btn ${i === 0 ? 'active' : ''}" onclick="selectEpisode('${animeId}', ${i}, this)">${i + 1}</button>`
  ).join('');
  document.getElementById('epList').innerHTML = epsHTML || '<p style="color:#666;font-size:13px">Пусто</p>';
  const frame = document.getElementById('playerFrame');
  if (frame && eps[0]) frame.src = eps[0];
}

function selectEpisode(animeId, epIndex, btn) {
  const a = getAnime(animeId);
  const activeVoice = [...document.querySelectorAll('.voice-btn')].findIndex(b => b.classList.contains('active'));
  const voiceName = a.voices[activeVoice]?.name || '';
  const src = a.voices[activeVoice]?.episodes[epIndex];
  document.querySelectorAll('.ep-btn').forEach(b => b.classList.remove('active'));
  if (btn) btn.classList.add('active');
  const frame = document.getElementById('playerFrame');
  if (frame && src) frame.src = src;

  // Сохранение истории
  if (CURRENT_USER) {
    api('/users/history', {
      method: 'POST',
      body: { animeId, episode: epIndex + 1, voice: voiceName }
    }).catch(() => {});
  }
}


async function toggleFavorite(animeId) {
  if (!CURRENT_USER) { openLogin(); return; }
  try {
    const res = await api('/users/favorites/' + animeId, { method: 'POST' });
    if (res.favorite) {
      if (!USER_FAVORITES.includes(animeId)) USER_FAVORITES.push(animeId);
      toast('Добавлено в избранное ♡');
    } else {
      USER_FAVORITES = USER_FAVORITES.filter(id => id !== animeId);
      toast('Удалено из избранного');
    }
    navigate('anime', animeId);
  } catch (e) {
    toast(e.message, true);
  }
}

// ---------- РЕЙТИНГ ----------
async function loadRating(animeId) {
  try {
    const { avg, count, my } = await api('/anime/' + animeId + '/rating');
    const avgEl = document.getElementById('avgRating');
    const cntEl = document.getElementById('ratingCount');
    if (avgEl) avgEl.textContent = avg ? avg.toFixed(1) : '—';
    if (cntEl) cntEl.textContent = count ? `${count} ${plural(count, 'оценка', 'оценки', 'оценок')}` : 'нет оценок';

    if (my) {
      document.querySelectorAll('#ratingStars button').forEach(b => {
        b.classList.toggle('active', +b.dataset.score <= my);
      });
    }
  } catch {}
}

async function setRating(animeId, score) {
  if (!CURRENT_USER) { openLogin(); return; }
  try {
    const res = await api('/anime/' + animeId + '/rating', {
      method: 'POST',
      body: { score }
    });
    document.querySelectorAll('#ratingStars button').forEach(b => {
      b.classList.toggle('active', +b.dataset.score <= score);
    });
    const avgEl = document.getElementById('avgRating');
    const cntEl = document.getElementById('ratingCount');
    if (avgEl) avgEl.textContent = res.avg.toFixed(1);
    if (cntEl) cntEl.textContent = `${res.count} ${plural(res.count, 'оценка', 'оценки', 'оценок')}`;
    toast('Оценка сохранена ♡');
  } catch (e) { toast(e.message, true); }
}

// ---------- КОММЕНТАРИИ ----------
async function loadComments(animeId) {
  try {
    const { comments } = await api('/anime/' + animeId + '/comments');
    const list = document.getElementById('commentsList');
    const cnt = document.getElementById('commentsCount');
    if (cnt) cnt.textContent = comments.length ? `(${comments.length})` : '';
    if (!list) return;
    if (!comments.length) {
      list.innerHTML = '<p style="color:var(--text-3);font-size:13px">Пока нет комментариев</p>';
      return;
    }
    list.innerHTML = comments.map(c => `
      <div class="comment-item">
        <div class="comment-head">
          <div class="comment-user">${escapeHtml(c.username)}</div>
          <div style="display:flex;gap:6px;align-items:center">
            <span class="comment-date">${timeAgo(c.created_at)}</span>
            ${CURRENT_USER && (CURRENT_USER.id === c.user_id || CURRENT_USER.role === 'admin')
              ? `<button class="comment-delete" onclick="deleteComment(${c.id}, '${animeId}')">✕</button>` : ''}
          </div>
        </div>
        <div class="comment-text">${escapeHtml(c.text)}</div>
      </div>
    `).join('');
  } catch (e) {
    const list = document.getElementById('commentsList');
    if (list) list.innerHTML = '<p style="color:var(--text-3);font-size:13px">Не удалось загрузить</p>';
  }
}

async function sendComment(animeId) {
  const ta = document.getElementById('commentText');
  const text = ta.value.trim();
  if (text.length < 2) { toast('Комментарий слишком короткий', true); return; }
  try {
    await api('/anime/' + animeId + '/comments', {
      method: 'POST',
      body: { text }
    });
    ta.value = '';
    toast('Комментарий добавлен ♡');
    loadComments(animeId);
  } catch (e) { toast(e.message, true); }
}

async function deleteComment(id, animeId) {
  if (!confirm('Удалить комментарий?')) return;
  try {
    await api('/comments/' + id, { method: 'DELETE' });
    loadComments(animeId);
  } catch (e) { toast(e.message, true); }
}

// ---------- ХЕЛПЕРЫ ----------
function plural(n, one, few, many) {
  const m10 = n % 10, m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20)) return few;
  return many;
}

function timeAgo(unixSeconds) {
  if (!unixSeconds) return 'только что';
  const now = Math.floor(Date.now() / 1000);
  const diff = now - unixSeconds;
  if (diff < 60) return 'только что';
  if (diff < 3600) return Math.floor(diff / 60) + ' мин назад';
  if (diff < 86400) return Math.floor(diff / 3600) + ' ч назад';
  if (diff < 604800) return Math.floor(diff / 86400) + ' дн назад';
  return new Date(unixSeconds * 1000).toLocaleDateString('ru-RU');
}

// ---------- СЛАЙДЕР ----------
let heroTimer = null;
function initHeroSlider() {
  const slides = document.querySelectorAll('.hero-slide');
  const dots = document.querySelectorAll('#heroDots span');
  if (!slides.length) return;
  let idx = 0;
  const show = (i) => {
    slides.forEach((s, k) => s.classList.toggle('active', k === i));
    dots.forEach((d, k) => d.classList.toggle('active', k === i));
    idx = i;
  };
  dots.forEach(d => d.addEventListener('click', () => show(+d.dataset.idx)));
  clearInterval(heroTimer);
  heroTimer = setInterval(() => show((idx + 1) % slides.length), 5000);
}

// ---------- ПОИСК ----------
function initSearch() {
  const input = document.getElementById('searchInput');
  const results = document.getElementById('searchResults');
  if (!input) return;

  input.addEventListener('input', () => {
    const q = input.value.trim().toLowerCase();
    if (!q) { results.classList.remove('active'); return; }
    const found = DATA.anime.filter(a =>
      a.title.toLowerCase().includes(q) ||
      (a.original || '').toLowerCase().includes(q)
    ).slice(0, 8);

    results.innerHTML = found.length
      ? found.map(a => `
        <div class="search-item" onclick="document.getElementById('searchInput').value='';document.getElementById('searchResults').classList.remove('active');navigate('anime','${a.id}')">
          <img src="${escapeHtml(a.poster)}" onerror="this.src='https://via.placeholder.com/40x56/FFDE00/000'">
          <div class="search-item-info">
            <h4>${escapeHtml(a.title)}</h4>
            <span>${a.year} · ${escapeHtml(a.genres.join(', '))}</span>
          </div>
        </div>`).join('')
      : `<div class="search-item"><div class="search-item-info"><h4>Ничего не найдено</h4></div></div>`;
    results.classList.add('active');
  });

  document.addEventListener('click', (e) => {
    if (!e.target.closest('.search-wrap')) results.classList.remove('active');
  });
}

// ---------- PARALLAX ФОНА ----------
function initParallax() {
  const chars = document.querySelectorAll('.chara');
  if (!chars.length) return;
  let mx = 0, my = 0, tx = 0, ty = 0;
  document.addEventListener('mousemove', (e) => {
    mx = (e.clientX / window.innerWidth - 0.5) * 2;
    my = (e.clientY / window.innerHeight - 0.5) * 2;
  });
  function loop() {
    tx += (mx - tx) * 0.05;
    ty += (my - ty) * 0.05;
    chars.forEach((c, i) => {
      const dir = i === 0 ? 1 : -1;
      c.style.transform = `translate(${tx * 12 * dir}px, ${ty * 8 * dir}px)`;
    });
    requestAnimationFrame(loop);
  }
  loop();
}

// ---------- СТАРТ ----------
document.addEventListener('DOMContentLoaded', async () => {
  initTheme();
  initSearch();
  initParallax();
  await loadAll();
  if (typeof initAuth === 'function') await initAuth();
  navigate('home');
});