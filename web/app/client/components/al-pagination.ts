export class AlPagination extends HTMLElement {
  static observedAttributes = ['page', 'total-pages'];

  private get page(): number {
    return parseInt(this.getAttribute('page') || '1', 10);
  }

  private get totalPages(): number {
    return parseInt(this.getAttribute('total-pages') || '1', 10);
  }

  connectedCallback() {
    this.render();
    this.addEventListener('click', this.handleClick);
  }

  disconnectedCallback() {
    this.removeEventListener('click', this.handleClick);
  }

  attributeChangedCallback() {
    this.render();
  }

  private handleClick = (e: Event) => {
    const target = e.target as HTMLElement;
    const button = target.closest('button[data-page]') as HTMLButtonElement;
    if (button && !button.disabled) {
      const page = parseInt(button.dataset.page!, 10);
      this.dispatchEvent(new CustomEvent('page-change', {
        detail: { page },
        bubbles: true,
      }));
    }
  };

  private render() {
    const { page, totalPages } = this;

    if (totalPages <= 1) {
      this.innerHTML = '';
      return;
    }

    this.innerHTML = `
      <div class="pagination">
        <button class="btn" data-page="${page - 1}" ${page <= 1 ? 'disabled' : ''}>
          Previous
        </button>
        <span class="pagination-info">Page ${page} of ${totalPages}</span>
        <button class="btn" data-page="${page + 1}" ${page >= totalPages ? 'disabled' : ''}>
          Next
        </button>
      </div>
    `;
  }
}

customElements.define('al-pagination', AlPagination);
