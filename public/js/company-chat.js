(function () {
  'use strict';

  const root = document.querySelector('[data-company-chat-root]');
  if (!root) return;
  const list = root.querySelector('[data-chat-messages]');
  if (!list) return;
  const compose = root.querySelector('#company-chat-message');
  const length = root.querySelector('[data-chat-length]');
  if (compose && length) {
    const updateLength = function () { length.textContent = String(Array.from(compose.value).length); };
    compose.addEventListener('input', updateLength);
    updateLength();
  }

  const moderator = root.dataset.moderator === 'true';
  const companyID = root.dataset.companyId;
  const csrf = root.dataset.csrf;
  const messagesURL = root.dataset.messagesUrl;
  let signature = '';

  function render(messages) {
    const nextSignature = messages.map(function (message) {
      return [message.id, message.moderated, message.message].join(':');
    }).join('|');
    if (nextSignature === signature) return;
    signature = nextSignature;

    const fragment = document.createDocumentFragment();
    if (!messages.length) {
      const empty = document.createElement('li');
      empty.className = 'company-chat-empty';
      empty.textContent = 'Aún no hay mensajes. Inicia la conversación.';
      fragment.appendChild(empty);
    }

    messages.forEach(function (message) {
      const item = document.createElement('li');
      item.className = 'company-chat-message' + (String(message.user_id) === root.dataset.userId ? ' company-chat-message--mine' : '') + (message.moderated ? ' company-chat-message--moderated' : '');
      item.dataset.messageId = String(message.id);

      const meta = document.createElement('div');
      meta.className = 'company-chat-message__meta';
      const avatar = message.user_photo ? document.createElement('img') : document.createElement('span');
      avatar.className = 'company-chat-message__avatar' + (message.user_photo ? '' : ' company-chat-message__avatar--fallback');
      if (message.user_photo) {
        avatar.src = message.user_photo;
        avatar.alt = '';
        avatar.loading = 'lazy';
      } else {
        const avatarIcon = document.createElement('i');
        avatarIcon.className = 'bi bi-person-fill';
        avatarIcon.setAttribute('aria-hidden', 'true');
        avatar.appendChild(avatarIcon);
      }
      const name = document.createElement('strong');
      name.textContent = message.user_name || 'Miembro';
      let identity = name;
      if (message.profile_url) {
        identity = document.createElement('a');
        identity.href = message.profile_url;
        identity.textContent = name.textContent;
      }
      const time = document.createElement('time');
      time.dateTime = message.created_at;
      const date = new Date(message.created_at);
      time.textContent = Number.isNaN(date.getTime()) ? '' : date.toLocaleString();
      meta.append(avatar, identity, time);

      const content = document.createElement('p');
      content.textContent = message.message || '';
      item.append(meta, content);

      if (moderator && !message.moderated) {
        const form = document.createElement('form');
        form.method = 'POST';
        form.action = '/empresas/' + encodeURIComponent(companyID) + '/chat/messages/' + encodeURIComponent(message.id) + '/moderate';
        form.addEventListener('submit', function (event) {
          if (!window.confirm('¿Retirar este mensaje del chat?')) event.preventDefault();
        });
        const token = document.createElement('input');
        token.type = 'hidden';
        token.name = '_csrf';
        token.value = csrf;
        const remove = document.createElement('button');
        remove.type = 'submit';
        remove.className = 'company-chat-message__moderate';
        remove.textContent = 'Retirar mensaje';
        form.append(token, remove);
        item.appendChild(form);
      }
      fragment.appendChild(item);
    });

    list.replaceChildren(fragment);
    list.scrollTop = list.scrollHeight;
  }

  async function refresh() {
    if (document.hidden) return;
    try {
      const response = await fetch(messagesURL, {
        credentials: 'same-origin',
        headers: { Accept: 'application/json' },
        cache: 'no-store'
      });
      if (!response.ok) return;
      const data = await response.json();
      if (Array.isArray(data.messages)) render(data.messages);
    } catch (_) {
      // Keep the last visible messages if the connection is temporarily lost.
    }
  }

  const initial = Array.from(list.querySelectorAll('[data-message-id]'));
  signature = initial.map(function (item) {
    return item.dataset.messageId + ':' + item.classList.contains('company-chat-message--moderated');
  }).join('|');
  list.scrollTop = list.scrollHeight;
  window.setInterval(refresh, 30000);
}());
