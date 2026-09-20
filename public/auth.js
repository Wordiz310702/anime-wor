/* ================= Авторизация ================= */

async function initAuth() {
  try {
    const { user } = await api('/users/me');
    CURRENT_USER = user;
    if (user.role === 'user' || user.role === 'admin') {
      const fav = await api('/users/favorites').catch(() => ({ favorites: [] }));
      USER_FAVORITES = fav.favorites || [];
    }
  } catch {
    CURRENT_USER = null;
  }
  renderUserArea();
}

function renderUserArea() {
  const el = document.getElementById('userArea');
  if (!el) return;

  const fl = document.getElementById('footerLogin');
  const fr = document.getElementById('footerRegister');

  if (CURRENT_USER) {
    const isAdmin = CURRENT_USER.role === 'admin';
    el.innerHTML = `
      <div class="user-badge ${isAdmin ? 'admin' : ''}">${escapeHtml(CURRENT_USER.username)}</div>
      ${isAdmin ? `<button class="btn btn-ghost btn-sm" onclick="navigate('admin')">Панель</button>` : ''}
      <button class="btn btn-ghost btn-sm" onclick="doLogout()">Выйти</button>
    `;
    if (fl) fl.style.display = 'none';
    if (fr) fr.style.display = 'none';
  } else {
    el.innerHTML = `
      <button class="btn btn-ghost btn-sm" onclick="openLogin()">Вход</button>
      <button class="btn btn-sm" onclick="openRegister()">Регистрация</button>
    `;
    if (fl) fl.style.display = '';
    if (fr) fr.style.display = '';
  }
}

function openLogin() {
  document.getElementById('loginModal').classList.add('active');
  setTimeout(() => document.getElementById('loginUser')?.focus(), 100);
}
function openRegister() {
  document.getElementById('registerModal').classList.add('active');
  setTimeout(() => document.getElementById('regUser')?.focus(), 100);
}
function closeModal(id) {
  document.getElementById(id).classList.remove('active');
}

async function doLogin() {
  const username = document.getElementById('loginUser').value.trim();
  const password = document.getElementById('loginPass').value;
  if (!username || !password) { toast('Заполните все поля', true); return; }

  try {
    const { user } = await api('/users/login', {
      method: 'POST',
      body: { username, password }
    });
    CURRENT_USER = user;
    closeModal('loginModal');
    document.getElementById('loginUser').value = '';
    document.getElementById('loginPass').value = '';

    const fav = await api('/users/favorites').catch(() => ({ favorites: [] }));
    USER_FAVORITES = fav.favorites || [];

    renderUserArea();
    toast(`Добро пожаловать, ${user.username}!`);
    navigate('home');
  } catch (e) {
    toast(e.message, true);
  }
}

async function doRegister() {
  const username = document.getElementById('regUser').value.trim();
  const email = document.getElementById('regEmail').value.trim();
  const password = document.getElementById('regPass').value;

  if (!username || !email || !password) { toast('Заполните все поля', true); return; }
  if (password.length < 6) { toast('Пароль минимум 6 символов', true); return; }

  try {
    const { user } = await api('/users/register', {
      method: 'POST',
      body: { username, email, password }
    });
    CURRENT_USER = user;
    closeModal('registerModal');
    document.getElementById('regUser').value = '';
    document.getElementById('regEmail').value = '';
    document.getElementById('regPass').value = '';

    USER_FAVORITES = [];
    renderUserArea();
    toast('Аккаунт создан!');
    navigate('home');
  } catch (e) {
    toast(e.message, true);
  }
}

async function doLogout() {
  try { await api('/users/logout', { method: 'POST' }); } catch {}
  CURRENT_USER = null;
  USER_FAVORITES = [];
  renderUserArea();
  toast('Вы вышли из аккаунта');
  navigate('home');
}

// Enter в модалках
document.addEventListener('keydown', (e) => {
  if (e.key === 'Enter') {
    if (document.getElementById('loginModal')?.classList.contains('active')) doLogin();
    if (document.getElementById('registerModal')?.classList.contains('active')) doRegister();
  }
  // Секретная комбинация Ctrl + Shift + A
  if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === 'a') {
    e.preventDefault();
    if (CURRENT_USER?.role === 'admin') navigate('admin');
    else if (!CURRENT_USER) openLogin();
    else toast('Доступ только для админа', true);
  }
});