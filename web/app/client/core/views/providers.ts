import { api, type PageResult, type PageRequest } from '../api';
import type { Provider, ProviderCommand } from '../domains';
import { html, escape } from '../html';
import { createView } from '../views';

createView('[data-view="providers"]', {
  onMount() {
    const tbody = document.querySelector<HTMLTableSectionElement>('#provider-tbody')!;
    const emptyState = document.querySelector<HTMLElement>('#provider-empty')!;
    const table = tbody.closest('table')!;
    const pagination = document.querySelector<HTMLElement>('#provider-pagination')!;
    const searchInput = document.querySelector<HTMLInputElement>('#provider-search')!;
    const addBtn = document.querySelector<HTMLButtonElement>('#add-provider-btn')!;
    const dialog = document.querySelector<HTMLDialogElement>('#provider-dialog')!;
    const dialogTitle = document.querySelector<HTMLElement>('#provider-dialog-title')!;
    const form = document.querySelector<HTMLFormElement>('#provider-form')!;
    const idInput = document.querySelector<HTMLInputElement>('#provider-id')!;
    const nameInput = document.querySelector<HTMLInputElement>('#provider-name')!;
    const configInput = document.querySelector<HTMLTextAreaElement>('#provider-config')!;
    const deleteBtn = document.querySelector<HTMLButtonElement>('#delete-provider-btn')!;
    const cancelBtn = document.querySelector<HTMLButtonElement>('#cancel-provider-btn')!;

    let currentPage = 1;
    let searchTimeout: number;

    function renderRows(providers: Provider[]): string {
      return providers.map(p => html`
        <tr data-id="${p.id}">
          <td>${p.name}</td>
          <td>${p.config?.type ?? '-'}</td>
          <td>${new Date(p.updated_at).toLocaleDateString()}</td>
        </tr>
      `).join('');
    }

    async function loadProviders(params: PageRequest = {}) {
      const result = await api.get<PageResult<Provider>>('/providers', {
        page: currentPage,
        search: searchInput.value || undefined,
        ...params,
      });

      if (!result.ok) {
        tbody.innerHTML = '';
        emptyState.textContent = `Error: ${result.error}`;
        emptyState.classList.remove('hidden');
        table.classList.add('hidden');
        return;
      }

      const { data, page, total_pages } = result.data;

      if (data.length === 0) {
        tbody.innerHTML = '';
        emptyState.classList.remove('hidden');
        table.classList.add('hidden');
      }
    }
  }
});
