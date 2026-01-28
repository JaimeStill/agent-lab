import { LitElement, html, unsafeCSS } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import styles from './not-found-view.css?inline';

@customElement('lab-not-found-view')
export class NotFoundView extends LitElement {
  static styles = unsafeCSS(styles);

  @property({ type: String }) path?: string;

  render() {
    return html`
      <div class="container">
        <h1>404</h1>
        <p>Page not found${this.path ? `: /${this.path}` : ''}</p>
        <a href="">Return home</a>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'lab-not-found-view': NotFoundView;
  }
}
