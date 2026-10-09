(() => {
  'use strict';
  if (document.getElementById('load-interest-dialog')) return;
  const triggers = document.querySelectorAll('[data-load-interest]');
  if (!triggers.length) return;
  const dialog = document.createElement('dialog');
  dialog.id = 'load-interest-dialog';
  dialog.className = 'bf-interest-dialog';
  dialog.setAttribute('aria-labelledby', 'load-interest-title');
  const form = document.createElement('form');
  form.method = 'post';
  const title = document.createElement('h2');
  title.id = 'load-interest-title';
  const description = document.createElement('p');
  description.textContent = 'Escribe un comentario para el publicador de esta carga.';
  const token = document.createElement('input');
  token.type = 'hidden'; token.name = '_csrf';
  const label = document.createElement('label');
  label.htmlFor = 'load-interest-comment'; label.textContent = 'Tu comentario';
  const comment = document.createElement('textarea');
  comment.id = label.htmlFor; comment.name = 'comment'; comment.required = true;
  comment.maxLength = 1000; comment.rows = 5; comment.className = 'form-control';
  const hint = document.createElement('p'); hint.textContent = 'Máximo 1000 caracteres.';
  const actions = document.createElement('div'); actions.className = 'bf-interest-actions';
  const cancel = document.createElement('button'); cancel.type = 'button';
  cancel.className = 'btn btn-secondary'; cancel.textContent = 'Cancelar';
  const send = document.createElement('button'); send.type = 'submit';
  send.className = 'btn btn-primary'; send.textContent = 'Enviar comentario';
  actions.append(cancel, send); form.append(title, description, token, label, comment, hint, actions);
  dialog.append(form); document.body.append(dialog);
  let origin;
  triggers.forEach(button => button.addEventListener('click', event => {
    event.preventDefault(); event.stopPropagation();
    if (!/^[1-9][0-9]*$/.test(button.dataset.loadInterest)) return;
    origin = button; form.action = '/loads/' + button.dataset.loadInterest + '/interest';
    title.textContent = 'Me interesa · ' + button.dataset.loadReference;
    token.value = button.dataset.csrf; comment.value = ''; comment.setCustomValidity('');
    send.disabled = false; dialog.showModal(); comment.focus();
  }));
  cancel.addEventListener('click', () => dialog.close());
  dialog.addEventListener('click', event => { if (event.target === dialog) {
    const bounds = dialog.getBoundingClientRect();
    if (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom) dialog.close();
  }});
  dialog.addEventListener('close', () => origin?.focus());
  comment.addEventListener('input', () => comment.setCustomValidity(''));
  form.addEventListener('submit', event => {
    if (!comment.value.trim()) { event.preventDefault(); comment.setCustomValidity('Escribe un comentario.'); comment.reportValidity(); return; }
    send.disabled = true;
  });
})();
