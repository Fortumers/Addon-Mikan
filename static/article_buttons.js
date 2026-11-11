// static/article_buttons.js
// Simple client‑side script for the "Кнопки статей" UI page.
// It reads the JSON data from the container element, renders a table
// with edit / delete controls, and updates the in‑memory data.
// In a real Mikan UI plugin you would send the updated data back to the
// server via the SDK's API (e.g., sdk.PostUpdate). Here we only provide the
// front‑end logic as a placeholder.

document.addEventListener('DOMContentLoaded', () => {
  const container = document.getElementById('article-buttons-app');
  if (!container) return;
  let buttons = [];
  try {
    buttons = JSON.parse(container.dataset.buttons);
  } catch (e) {
    console.error('Failed to parse buttons JSON', e);
  }

  const render = () => {
    container.innerHTML = '';
    const table = document.createElement('table');
    table.style.borderCollapse = 'collapse';
    const header = document.createElement('tr');
    ['ID', 'Label', 'Emoji', 'Callback', 'Disabled', 'Selected', 'Color', 'Actions'].forEach(col => {
      const th = document.createElement('th');
      th.textContent = col;
      th.style.border = '1px solid #ccc';
      th.style.padding = '4px 8px';
      header.appendChild(th);
    });
    table.appendChild(header);

    buttons.forEach((b, idx) => {
      const tr = document.createElement('tr');
      const cells = [
        b.id || idx,
        b.label,
        b.emoji || '-',
        b.callback_data,
        b.disabled ? '✓' : '',
        b.selected ? '✓' : '',
        b.color_hex || '-',
      ];
      cells.forEach(text => {
        const td = document.createElement('td');
        td.textContent = text;
        td.style.border = '1px solid #eee';
        td.style.padding = '4px 8px';
        tr.appendChild(td);
      });
      // Actions column
      const actionsTd = document.createElement('td');
      const editBtn = document.createElement('button');
      editBtn.textContent = 'Edit';
      editBtn.style.marginRight = '4px';
      editBtn.onclick = () => editButton(idx);
      const delBtn = document.createElement('button');
      delBtn.textContent = 'Delete';
      delBtn.onclick = () => deleteButton(idx);
      actionsTd.appendChild(editBtn);
      actionsTd.appendChild(delBtn);
      actionsTd.style.border = '1px solid #eee';
      actionsTd.style.padding = '4px 8px';
      tr.appendChild(actionsTd);

      table.appendChild(tr);
    });

    // Add new button form
    const form = document.createElement('div');
    form.style.marginTop = '12px';
    form.innerHTML = `
      <h3>Add new button</h3>
      <label>Label: <input id="new-label" type="text"></label>
      <label> Emoji: <input id="new-emoji" type="text" placeholder="🤖"></label>
      <label> Callback: <input id="new-callback" type="text"></label>
      <label> Disabled: <input id="new-disabled" type="checkbox"></label>
      <label> Selected: <input id="new-selected" type="checkbox"></label>
      <label> Color (HEX): <input id="new-color" type="text" placeholder="#ff0000"></label>
      <button id="add-button">Add</button>
    `;
    form.querySelector('#add-button').addEventListener('click', addButton);
    container.appendChild(table);
    container.appendChild(form);
  };

  const addButton = () => {
    const label = document.getElementById('new-label').value.trim();
    const emoji = document.getElementById('new-emoji').value.trim();
    const callback = document.getElementById('new-callback').value.trim();
    const disabled = document.getElementById('new-disabled').checked;
    const selected = document.getElementById('new-selected').checked;
    const color = document.getElementById('new-color').value.trim();
    if (!label || !callback) {
      alert('Label and Callback are required');
      return;
    }
    const newBtn = {
      id: Date.now().toString(),
      label,
      emoji,
      callback_data: callback,
      disabled,
      selected,
      color_hex: color,
    };
    buttons.push(newBtn);
    render();
    // TODO: send updated list to backend via SDK API
  };

  const editButton = (index) => {
    const b = buttons[index];
    const label = prompt('Edit label', b.label);
    if (label !== null) b.label = label;
    const emoji = prompt('Edit emoji', b.emoji || '');
    if (emoji !== null) b.emoji = emoji;
    const callback = prompt('Edit callback', b.callback_data);
    if (callback !== null) b.callback_data = callback;
    const disabled = confirm('Disabled?');
    b.disabled = disabled;
    const selected = confirm('Selected?');
    b.selected = selected;
    const color = prompt('Edit color HEX', b.color_hex || '');
    if (color !== null) b.color_hex = color;
    render();
    // TODO: persist changes via SDK
  };

  const deleteButton = (index) => {
    if (!confirm('Delete this button?')) return;
    buttons.splice(index, 1);
    render();
    // TODO: persist deletion via SDK
  };

  render();
});
