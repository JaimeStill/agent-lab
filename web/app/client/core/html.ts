const escapeMap: Record<string, string> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
}

export function escape(value: unknown): string {
  const str = String(value ?? '');
  return str.replace(/[&<>"']/g, (char) => escapeMap[char]);
}

export function html(
  strings: TemplateStringsArray,
  ...values: unknown[]
): string {
  return strings.reduce((result, str, i) => {
    const value = i < values.length ? escape(values[i]) : '';
    return result + str + value;
  }, '');
}
