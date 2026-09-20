/* ================= Админ-панель ================= */

let currentTab = 'anime';
let voiceRows = [];

function renderAdmin() {
  if (!CURRENT_USER || CURRENT_USER.role !== 'admin') {
    return `<div class="container"><div class="empty"><h3>Доступ запрещён</h3></div></div>`;
  }

  return `<div class="container admin-page">
    <section class="section" style="margin-top:0">
      <div class="section-head"><h2>Админ-панель</h2></div>
      <div class="admin-tabs">
        <button class="admin-tab ${currentTab === 'dashboard' ? 'active' : ''}" data-tab="dashboard" onclick="switchTab('dashboard')">📊 Дашборд</button>
        <button class="admin-tab ${currentTab === 'anime' ? 'active' : ''}" data-tab="anime" onclick="switchTab('anime')">📺 Аниме</button>
        <button class="admin-tab ${currentTab === 'banners' ? 'active' : ''}" data-tab="banners" onclick="switchTab('banners')">🖼 Баннеры</button>
        <button class="admin-tab ${currentTab === 'schedule' ? 'active' : ''}" data-tab="schedule" onclick="switchTab('schedule')">📅 Расписание</button>
        <button class="admin-tab ${currentTab === 'comments' ? 'active' : ''}" data-tab="comments" onclick="switchTab('comments')">💬 Комментарии</button>
        <button class="admin-tab ${currentTab === 'gallery' ? 'active' : ''}" data-tab="gallery" onclick="switchTab('gallery')">🗂 Файлы</button>
      </div>
      <div id="tabContent"></div>
    </section>
  </div>`;
}

async function switchTab(tab) {
  currentTab = tab;
  document.querySelectorAll('.admin-tab').forEach(b => {
    b.classList.toggle('active', b.dataset.tab === tab);
  });
  await renderTabContent();
}

async function renderTabContent() {
  const el = document.getElementById('tabContent');
  if (!el) return;

  // Показываем индикатор загрузки на время запросов
  el.innerHTML = '<div style="padding:40px;color:var(--text-3);font-size:14px">Загрузка...</div>';

  try {
    if (currentTab === 'dashboard') {
      el.innerHTML = await tabDashboard();
    } else if (currentTab === 'anime') {
      el.innerHTML = tabAnime();
    } else if (currentTab === 'banners') {
      el.innerHTML = tabBanners();
    } else if (currentTab === 'schedule') {
      el.innerHTML = tabSchedule();
      await loadSchedule();
    } else if (currentTab === 'comments') {
      el.innerHTML = await tabComments();
    } else if (currentTab === 'gallery') {
      el.innerHTML = tabGallery();
      await loadGallery();
    }
  } catch (e) {
    console.error('[renderTabContent] Ошибка:', e);
    el.innerHTML = `<div class="panel"><h3>Ошибка</h3><p style="color:#E5405E">${e.message}</p></div>`;
  }
}

/* ---------- АНИМЕ ---------- */
function tabAnime() {
  const list = DATA.anime.map(a => `
    <div class="list-item">
      <div class="list-item-info">
        <img src="${escapeHtml(a.poster)}" onerror="this.src='https://via.placeholder.com/46x64/FF6B35/fff'">
        <div>
          <h4>${escapeHtml(a.title)}</h4>
          <span>${a.year} · ${escapeHtml(a.genres.join(', '))} · Озвучек: ${a.voices.length}</span>
        </div>
      </div>
      <div style="display:flex; gap:6px; flex-shrink:0">
        <button class="btn btn-ghost btn-sm" onclick="editAnime('${a.id}')">✎</button>
        <button class="btn btn-danger btn-sm" onclick="deleteAnime('${a.id}')">✕</button>
      </div>
    </div>`).join('') || '<div class="empty"><h3>Пусто</h3></div>';

  return `
    <div class="panel">
      <h3>Добавить / изменить аниме</h3>
      <div class="form-grid">
        <input type="hidden" id="af_id">
        <div class="field"><label>Название *</label><input id="af_title" placeholder="Например: Наруто"></div>
        <div class="field"><label>Оригинальное</label><input id="af_original" placeholder="Naruto"></div>
        <div class="field"><label>Год</label><input id="af_year" type="number" value="2024"></div>
        <div class="field"><label>Рейтинг</label><input id="af_rating" type="number" step="0.1" value="8.0"></div>
        <div class="field"><label>Статус</label>
          <select id="af_status">
            <option>Завершён</option>
            <option>Анонс</option>
          </select>
        </div>
        <div class="field"><label>Жанры (через запятую)</label><input id="af_genres" placeholder="Экшен, Приключения, Сёнэн"></div>

        <div class="field full">
          <label>Постер</label>
          <div style="display:flex; gap:10px; flex-wrap:wrap; align-items:center">
            <input type="file" id="af_poster_file" accept="image/*" style="padding:8px; max-width:280px">
            <span style="color:var(--text-3); font-size:12px">— или —</span>
            <input id="af_poster" placeholder="Вставьте ссылку на картинку" style="flex:1; min-width:220px">
          </div>
          <div id="af_poster_preview" style="margin-top:10px"></div>
        </div>

        <div class="field full"><label>Описание *</label><textarea id="af_desc" placeholder="Краткое описание сюжета..."></textarea></div>
      </div>

      <div class="panel" style="margin-top:16px; background:rgba(0,0,0,0.25); box-shadow:none">
        <h3>Озвучки и эпизоды</h3>
        <div id="voicesEditor"></div>
        <button class="btn btn-ghost btn-sm" style="margin-top:10px" onclick="addVoiceRow()">+ Добавить озвучку</button>
      </div>

      <div style="margin-top:16px; display:flex; gap:8px; flex-wrap:wrap">
        <button class="btn btn-success" onclick="saveAnime()">Сохранить</button>
        <button class="btn btn-ghost" onclick="resetAnimeForm()">Очистить</button>
      </div>
    </div>

    <div class="panel">
      <h3>Список аниме (${DATA.anime.length})</h3>
      ${list}
    </div>`;
}

function addVoiceRow(name = '', episodes = '') {
  voiceRows.push({ name, episodes });
  renderVoiceRows();
}

function renderVoiceRows() {
  const el = document.getElementById('voicesEditor');
  if (!el) return;
  if (!voiceRows.length) {
    el.innerHTML = '<p style="font-size:13px;color:#666">Озвучек нет</p>';
    return;
  }
  el.innerHTML = voiceRows.map((v, i) => `
    <div class="form-grid" style="margin-bottom:12px; padding:12px; background:var(--white); border:2.5px solid var(--black); border-radius:var(--radius)">
      <div class="field"><label>Название</label>
        <input value="${escapeHtml(v.name)}" oninput="voiceRows[${i}].name=this.value">
      </div>
      <div class="field"><label>Ссылки на серии через запятую</label>
        <input value="${escapeHtml(v.episodes)}" oninput="voiceRows[${i}].episodes=this.value">
      </div>
      <div class="full" style="text-align:right">
        <button class="btn btn-danger btn-sm" onclick="voiceRows.splice(${i},1);renderVoiceRows()">Удалить</button>
      </div>
    </div>`).join('');
}

function resetAnimeForm() {
  ['af_id','af_title','af_original','af_genres','af_poster','af_desc'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.value = '';
  });
  const file = document.getElementById('af_poster_file');
  if (file) file.value = '';
  const prev = document.getElementById('af_poster_preview');
  if (prev) prev.innerHTML = '';
  const yearEl = document.getElementById('af_year');
  if (yearEl) yearEl.value = 2024;
  const ratingEl = document.getElementById('af_rating');
  if (ratingEl) ratingEl.value = 8.0;
  const statusEl = document.getElementById('af_status');
  if (statusEl) statusEl.value = 'Завершён';
  voiceRows = [];
  renderVoiceRows();
}

async function saveAnime() {
  const title = document.getElementById('af_title').value.trim();
  const description = document.getElementById('af_desc').value.trim();
  if (!title || !description) { toast('Заполните название и описание', true); return; }
  if (!voiceRows.length) { toast('Добавьте хотя бы одну озвучку', true); return; }

  // Определяем постер: сначала файл, потом URL
  let poster = document.getElementById('af_poster').value.trim();
  const fileInput = document.getElementById('af_poster_file');
  const file = fileInput?.files?.[0];

  if (file) {
    try {
      toast('Загружаем постер...');
      poster = await uploadFile(file, 'poster');
    } catch (e) {
      toast('Ошибка загрузки: ' + e.message, true);
      return;
    }
  }

  if (!poster) { toast('Загрузите файл или вставьте ссылку на постер', true); return; }

  const voices = voiceRows.map(v => ({
    name: v.name.trim() || 'Без названия',
    episodes: v.episodes.split(',').map(s => s.trim()).filter(Boolean)
  }));

  const payload = {
    title,
    original: document.getElementById('af_original').value.trim(),
    year: +document.getElementById('af_year').value || 2024,
    rating: +document.getElementById('af_rating').value || 0,
    status: document.getElementById('af_status').value,
    genres: document.getElementById('af_genres').value.split(',').map(s => s.trim()).filter(Boolean),
    poster,
    description,
    voices
  };

  const id = document.getElementById('af_id').value;
  try {
    if (id) {
      await api('/anime/' + id, { method: 'PUT', body: payload });
      toast('Аниме обновлено');
    } else {
      await api('/anime', { method: 'POST', body: payload });
      toast('Аниме добавлено');
    }
    await loadAll();
    resetAnimeForm();
    renderTabContent();
  } catch (e) { toast(e.message, true); }
  await loadAll();
  resetAnimeForm();
  await renderTabContent();   // ← добавить await
  toast('Аниме добавлено ♡');  
}

function editAnime(id) {
  const a = getAnime(id);
  if (!a) return;
  document.getElementById('af_id').value = a.id;
  document.getElementById('af_title').value = a.title;
  document.getElementById('af_original').value = a.original || '';
  document.getElementById('af_year').value = a.year;
  document.getElementById('af_rating').value = a.rating;
  document.getElementById('af_status').value = a.status;
  document.getElementById('af_genres').value = a.genres.join(', ');
  document.getElementById('af_poster').value = a.poster;
  document.getElementById('af_desc').value = a.description;
  voiceRows = a.voices.map(v => ({ name: v.name, episodes: v.episodes.join(', ') }));
  renderVoiceRows();
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

async function deleteAnime(id) {
  if (!confirm('Удалить это аниме?')) return;
  try {
    await api('/anime/' + id, { method: 'DELETE' });
    await loadAll();
    toast('Удалено');
    renderTabContent();
  } catch (e) { toast(e.message, true); }
}

/* ---------- БАННЕРЫ ---------- */
/* ---------- БАННЕРЫ ---------- */
function tabBanners() {
  const list = DATA.banners.map(b => `
    <div class="list-item">
      <div class="list-item-info">
        <img src="${escapeHtml(b.image)}" onerror="this.src='https://via.placeholder.com/46x64/FF6B35/fff'">
        <div>
          <h4>${escapeHtml(b.title)}</h4>
          <span>${escapeHtml(b.subtitle || '—')}</span>
        </div>
      </div>
      <button class="btn btn-danger btn-sm" onclick="deleteBanner('${b.id}')">✕</button>
    </div>`).join('') || '<div class="empty"><h3>Нет баннеров</h3><p>Добавь первый баннер</p></div>';

  const animeOptions = DATA.anime.map(a =>
    `<option value="${a.id}">${escapeHtml(a.title)}</option>`
  ).join('');

  return `
    <div class="panel">
      <h3>Добавить баннер</h3>
      <div class="form-grid">
        <div class="field"><label>Заголовок *</label><input id="bf_title" placeholder="Например: Атака титанов"></div>
        <div class="field"><label>Подзаголовок</label><input id="bf_subtitle" placeholder="Краткое описание"></div>

        <div class="field full">
          <label>Изображение баннера *</label>
          <div style="display:flex; gap:10px; flex-wrap:wrap; align-items:center">
            <input type="file" id="bf_image_file" accept="image/*" style="padding:8px; max-width:280px">
            <span style="color:var(--text-3); font-size:12px">— или —</span>
            <input id="bf_image" placeholder="Вставьте ссылку на картинку" style="flex:1; min-width:220px">
          </div>
          <div id="bf_image_preview" style="margin-top:10px"></div>
        </div>

        <div class="field"><label>Привязать к аниме</label>
          <select id="bf_anime">${animeOptions || '<option value="">Сначала добавь аниме</option>'}</select>
        </div>
      </div>

      <div style="margin-top:16px; display:flex; gap:8px; flex-wrap:wrap">
        <button class="btn btn-success" onclick="saveBanner()">Сохранить баннер</button>
        <button class="btn btn-ghost" onclick="resetBannerForm()">Очистить</button>
      </div>
    </div>

    <div class="panel">
      <h3>Список баннеров (${DATA.banners.length})</h3>
      ${list}
    </div>`;
}

async function saveBanner() {
  const title = document.getElementById('bf_title').value.trim();
  const subtitle = document.getElementById('bf_subtitle').value.trim();
  const animeId = document.getElementById('bf_anime').value;
  if (!title) { toast('Введите заголовок', true); return; }

  let image = document.getElementById('bf_image').value.trim();
  const file = document.getElementById('bf_image_file')?.files?.[0];

  if (file) {
    try {
      toast('Загружаем изображение...');
      poster = await uploadFile(file, 'poster');
    } catch (e) {
      toast('Ошибка загрузки: ' + e.message, true);
      return;
    }
  }
  if (!image) { toast('Загрузите файл или вставьте ссылку', true); return; }

  try {
    await api('/banners', {
      method: 'POST',
      body: { title, subtitle, image, animeId }
    });
    await loadAll();
    resetBannerForm();
    toast('Баннер добавлен ♡');
    renderTabContent();
  } catch (e) {
    toast(e.message, true);
  }
  await loadAll();
  resetBannerForm();
  await renderTabContent();   // ← добавить await
  toast('Баннер добавлен ♡');  
}

function resetBannerForm() {
  ['bf_title', 'bf_subtitle', 'bf_image'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.value = '';
  });
  const file = document.getElementById('bf_image_file');
  if (file) file.value = '';
  const prev = document.getElementById('bf_image_preview');
  if (prev) prev.innerHTML = '';
}

async function uploadFile(file, kind = 'poster') {
  const fd = new FormData();
  fd.append('file', file);
  const res = await fetch('/api/upload?kind=' + kind, {
    method: 'POST',
    body: fd,
    credentials: 'same-origin'
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || 'Ошибка загрузки');
  return data.url;
}

function resetBannerForm() {
  ['bf_title', 'bf_subtitle', 'bf_image'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.value = '';
  });
  const file = document.getElementById('bf_image_file');
  if (file) file.value = '';
  const prev = document.getElementById('bf_image_preview');
  if (prev) prev.innerHTML = '';
}

// Отправка файла на сервер Go → возвращает URL
async function uploadFile(file) {
  const fd = new FormData();
  fd.append('file', file);

  const res = await fetch('/api/upload', {
    method: 'POST',
    body: fd,
    credentials: 'same-origin'
  });

  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || 'Ошибка загрузки');
  return data.url;
}

async function deleteBanner(id) {
  if (!confirm('Удалить баннер?')) return;
  try {
    await api('/banners/' + id, { method: 'DELETE' });
    await loadAll();
    toast('Удалено');
    renderTabContent();
  } catch (e) { toast(e.message, true); }
}

async function initAdmin() {
  voiceRows = [];
  currentTab = 'dashboard';
  await renderTabContent();
}

// Живой предпросмотр выбранных файлов
document.addEventListener('change', (e) => {
  // Постер аниме
  if (e.target.id === 'af_poster_file') {
    const prev = document.getElementById('af_poster_preview');
    const file = e.target.files[0];
    if (prev && file) {
      const url = URL.createObjectURL(file);
      prev.innerHTML = `
        <div style="display:flex; align-items:center; gap:12px">
          <img src="${url}" style="max-height:160px; border-radius:12px; border:1px solid rgba(255,255,255,.1)">
          <div style="font-size:12px; color:var(--text-3)">
            <b style="color:var(--text)">${escapeHtml(file.name)}</b><br>
            ${(file.size / 1024).toFixed(0)} КБ
          </div>
        </div>`;
    }
  }

  // Баннер
  if (e.target.id === 'bf_image_file') {
    const prev = document.getElementById('bf_image_preview');
    const file = e.target.files[0];
    if (prev && file) {
      const url = URL.createObjectURL(file);
      prev.innerHTML = `
        <div style="display:flex; align-items:center; gap:12px">
          <img src="${url}" style="max-height:120px; max-width:280px; object-fit:cover; border-radius:12px; border:1px solid rgba(255,255,255,.1)">
          <div style="font-size:12px; color:var(--text-3)">
            <b style="color:var(--text)">${escapeHtml(file.name)}</b><br>
            ${(file.size / 1024).toFixed(0)} КБ
          </div>
        </div>`;
    }
  }
});

/* ---------- ДАШБОРД ---------- */
async function tabDashboard() {
  try {
    const data = await api('/admin/stats');
    const c = data.counts;
    return `
      <div class="panel">
        <h3>Общая статистика</h3>
        <div class="stats-grid">
          <div class="stat-card"><div class="num">${c.users}</div><div class="lbl">Пользователей</div></div>
          <div class="stat-card"><div class="num">${c.anime}</div><div class="lbl">Аниме</div></div>
          <div class="stat-card"><div class="num">${c.banners}</div><div class="lbl">Баннеров</div></div>
          <div class="stat-card"><div class="num">${c.comments}</div><div class="lbl">Комментариев</div></div>
          <div class="stat-card"><div class="num">${c.favorites}</div><div class="lbl">В избранном</div></div>
          <div class="stat-card"><div class="num">${c.ratings}</div><div class="lbl">Оценок</div></div>
          <div class="stat-card"><div class="num">${c.schedules}</div><div class="lbl">В расписании</div></div>
        </div>
      </div>

      <div class="panel">
        <h3>Топ по просмотрам</h3>
        ${data.top_views.length
          ? data.top_views.map(v => `
              <div class="list-item">
                <div class="list-item-info">
                  <img src="${escapeHtml(v.poster || '')}" onerror="this.src='https://via.placeholder.com/48x66/FF6B35/fff'">
                  <div>
                    <h4>${escapeHtml(v.title || 'Удалено')}</h4>
                    <span>Просмотров: ${v.views}</span>
                  </div>
                </div>
              </div>`).join('')
          : '<p style="color:var(--text-3);font-size:13px">Ещё нет данных</p>'}
      </div>

      <div class="panel">
        <h3>Топ по оценкам</h3>
        ${data.top_rated.length
          ? data.top_rated.map(v => `
              <div class="list-item">
                <div class="list-item-info">
                  <img src="${escapeHtml(v.poster || '')}" onerror="this.src='https://via.placeholder.com/48x66/FF6B35/fff'">
                  <div>
                    <h4>${escapeHtml(v.title || 'Удалено')}</h4>
                    <span>Средняя: ${v.avg.toFixed(1)} · Оценок: ${v.count}</span>
                  </div>
                </div>
              </div>`).join('')
          : '<p style="color:var(--text-3);font-size:13px">Ещё нет оценок</p>'}
      </div>`;
  } catch (e) {
    return `<div class="panel"><h3>Ошибка загрузки</h3><p>${escapeHtml(e.message)}</p></div>`;
  }
}

/* ---------- РАСПИСАНИЕ ---------- */
function tabSchedule() {
  const animeOptions = DATA.anime.map(a =>
    `<option value="${a.id}">${escapeHtml(a.title)}</option>`
  ).join('');

  return `
    <div class="panel">
      <h3>Добавить в расписание</h3>
      <div class="form-grid">
        <div class="field"><label>Аниме</label>
          <select id="sch_anime">${animeOptions || '<option>Сначала добавьте аниме</option>'}</select>
        </div>
        <div class="field"><label>День недели</label>
          <select id="sch_weekday">
            <option value="1">Понедельник</option>
            <option value="2">Вторник</option>
            <option value="3">Среда</option>
            <option value="4">Четверг</option>
            <option value="5">Пятница</option>
            <option value="6">Суббота</option>
            <option value="7">Воскресенье</option>
          </select>
        </div>
        <div class="field"><label>Номер серии</label><input id="sch_ep" type="number" value="1"></div>
        <div class="field"><label>Время (например 19:30)</label><input id="sch_time" placeholder="19:30"></div>
      </div>
      <div style="margin-top:16px">
        <button class="btn btn-success" onclick="saveSchedule()">Добавить</button>
      </div>
    </div>

    <div class="panel">
      <h3>Текущее расписание</h3>
      <div id="scheduleList"><p style="color:var(--text-3);font-size:13px">Загрузка...</p></div>
    </div>`;
}

async function loadSchedule() {
  try {
    const { schedule } = await api('/schedule');
    const el = document.getElementById('scheduleList');
    if (!el) return;
    if (!schedule.length) {
      el.innerHTML = '<p style="color:var(--text-3);font-size:13px">Пусто</p>';
      return;
    }
    const days = ['', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
    el.innerHTML = schedule.map(s => `
      <div class="list-item">
        <div class="list-item-info">
          <img src="${escapeHtml(s.poster || '')}" onerror="this.src='https://via.placeholder.com/48x66/FF6B35/fff'">
          <div>
            <h4>${escapeHtml(s.title || 'Удалено')}</h4>
            <span>${days[s.weekday]} · Серия ${s.episode || '—'} · ${s.air_time || '—'}</span>
          </div>
        </div>
        <button class="btn btn-danger btn-sm" onclick="deleteSchedule(${s.id})">✕</button>
      </div>
    `).join('');
  } catch (e) {}
}

async function saveSchedule() {
  const body = {
    animeId: document.getElementById('sch_anime').value,
    weekday: +document.getElementById('sch_weekday').value,
    episode: +document.getElementById('sch_ep').value,
    airTime: document.getElementById('sch_time').value.trim()
  };
  try {
    await api('/schedule', { method: 'POST', body });
    toast('Добавлено в расписание');
    loadSchedule();
  } catch (e) { toast(e.message, true); }
}

async function deleteSchedule(id) {
  if (!confirm('Удалить запись?')) return;
  try {
    await api('/schedule/' + id, { method: 'DELETE' });
    loadSchedule();
  } catch (e) { toast(e.message, true); }
}

/* ---------- КОММЕНТАРИИ ---------- */
async function tabComments() {
  try {
    const { comments } = await api('/admin/comments');
    return `
      <div class="panel">
        <h3>Все комментарии (${comments.length})</h3>
        ${comments.length
          ? comments.map(c => `
              <div class="list-item" style="align-items:flex-start">
                <div class="list-item-info" style="flex-direction:column; align-items:flex-start; gap:6px">
                  <div style="display:flex;gap:10px;align-items:center;flex-wrap:wrap">
                    <b style="color:var(--accent)">${escapeHtml(c.username)}</b>
                    <span style="font-size:12px;color:var(--text-3)">→ ${escapeHtml(c.anime_id)}</span>
                    ${c.approved ? '' : '<span style="font-size:11px;color:#E5405E">скрыт</span>'}
                  </div>
                  <div style="font-size:14px;color:var(--text-2)">${escapeHtml(c.text)}</div>
                </div>
                <div style="display:flex;gap:6px;flex-shrink:0">
                  <button class="btn btn-ghost btn-sm" onclick="toggleComment(${c.id}, ${c.approved ? 0 : 1})">${c.approved ? '🙈' : '👁'}</button>
                  <button class="btn btn-danger btn-sm" onclick="adminDeleteComment(${c.id})">✕</button>
                </div>
              </div>`).join('')
          : '<p style="color:var(--text-3);font-size:13px">Пока нет комментариев</p>'}
      </div>`;
  } catch (e) {
    return `<div class="panel"><h3>Ошибка</h3><p>${escapeHtml(e.message)}</p></div>`;
  }
}

async function toggleComment(id, approved) {
  try {
    await api('/admin/comments/' + id + '/approve', {
      method: 'POST',
      body: { approved }
    });
    renderTabContent();
  } catch (e) { toast(e.message, true); }
}

async function adminDeleteComment(id) {
  if (!confirm('Удалить комментарий?')) return;
  try {
    await api('/comments/' + id, { method: 'DELETE' });
    renderTabContent();
  } catch (e) { toast(e.message, true); }
}

/* ---------- ГАЛЕРЕЯ ФАЙЛОВ ---------- */
function tabGallery() {
  return `
    <div class="panel">
      <h3>Загруженные файлы</h3>
      <p style="color:var(--text-3);font-size:13px;margin-bottom:16px">
        Нажмите на файл, чтобы скопировать его URL. Так можно переиспользовать уже загруженные картинки.
      </p>
      <div class="gallery" id="galleryList">
        <p style="color:var(--text-3);font-size:13px">Загрузка...</p>
      </div>
    </div>`;
}

async function loadGallery() {
  try {
    const { files } = await api('/upload/list');
    const el = document.getElementById('galleryList');
    if (!el) return;
    if (!files.length) {
      el.innerHTML = '<p style="color:var(--text-3);font-size:13px">Файлов нет</p>';
      return;
    }
    el.innerHTML = files
      .sort((a, b) => b.mod_time - a.mod_time)
      .map(f => {
        const isImg = /\.(jpg|jpeg|png|webp|gif)$/i.test(f.name);
        return `
          <div class="gallery-item" onclick="copyFileUrl('${f.url}')" title="${f.name}">
            ${isImg
              ? `<img src="${f.url}" loading="lazy" onerror="this.style.display='none'">`
              : `<div style="display:grid;place-items:center;height:100%;font-size:36px">📹</div>`}
            <div class="gallery-actions">
              <button onclick="event.stopPropagation();deleteFile('${f.name}')" title="Удалить">✕</button>
            </div>
          </div>`;
      }).join('');
  } catch (e) {
    const el = document.getElementById('galleryList');
    if (el) el.innerHTML = `<p style="color:#E5405E;font-size:13px">${escapeHtml(e.message)}</p>`;
  }
}

async function copyFileUrl(url) {
  try {
    const full = location.origin + url;
    await navigator.clipboard.writeText(full);
    toast('Ссылка скопирована: ' + url);
  } catch {
    toast('URL: ' + url);
  }
}

async function deleteFile(name) {
  if (!confirm('Удалить файл "' + name + '"?')) return;
  try {
    await api('/upload/' + encodeURIComponent(name), { method: 'DELETE' });
    toast('Удалено');
    loadGallery();
  } catch (e) { toast(e.message, true); }
}