import { LitElement, html, unsafeCSS } from 'lit';
import { customElement } from 'lit/decorators.js';
import styles from './home-view.css?inline';

@customElement('lab-home-view')
export class HomeView extends LitElement {
  static styles = unsafeCSS(styles);

  render() {
    return html`
      <div class="container">
        <h1>Agent Lab</h1>
        <p>Workflow execution and monitoring interface.</p>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'lab-home-view': HomeView;
  }
}
