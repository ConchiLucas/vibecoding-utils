const formatSqlCommentBlock = (text: string) => {
  return String(text || '')
    .trim()
    .split(/\r?\n/)
    .map((line) => (line.trim() ? `-- ${line}` : '--'))
    .join('\n');
};

export type DbTemplatePlaceholder = {
  key: string;
  description: string;
  value: string;
};

export type DbTemplatePlaceholderValues = Record<string, string>;

const normalizePlaceholderKey = (value: any) => {
  let key = String(value || '').trim();
  const mustacheMatch = key.match(/^\{\{\s*(.+?)\s*\}\}$/);
  if (mustacheMatch) {
    key = mustacheMatch[1];
  }
  const dollarMatch = key.match(/^\$\{\s*(.+?)\s*\}$/);
  if (dollarMatch) {
    key = dollarMatch[1];
  }
  const bracketMatch = key.match(/^\{\[\s*<\s*(.+?)\s*>\s*\]\}$/);
  if (bracketMatch) {
    key = bracketMatch[1];
  }
  const scopedMatch = key.match(/^(?:manual|field|snippet|parsed)\s*:\s*(.+)$/i);
  if (scopedMatch) {
    key = scopedMatch[1];
  }
  const angleMatch = key.match(/^<\s*(.+?)\s*>$/);
  if (angleMatch) {
    key = angleMatch[1];
  }
  return key.trim();
};

const normalizePlaceholder = (item: any): DbTemplatePlaceholder => ({
  key: normalizePlaceholderKey(item?.key),
  description: String(item?.description || '').trim(),
  value: String(item?.value || '').trim(),
});

export const parseDbTemplatePlaceholders = (value: any): DbTemplatePlaceholder[] => {
  if (Array.isArray(value)) {
    return value.map(normalizePlaceholder).filter((item) => item.key);
  }
  const raw = String(value || '').trim();
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.map(normalizePlaceholder).filter((item) => item.key) : [];
  } catch (e) {
    return [];
  }
};

export const stringifyDbTemplatePlaceholders = (value: any): string => {
  const list = parseDbTemplatePlaceholders(value);
  return list.length > 0 ? JSON.stringify(list) : '';
};

export const mergeDbTemplatePlaceholders = (types: any[]): DbTemplatePlaceholder[] => {
  const merged = new Map<string, DbTemplatePlaceholder>();
  (Array.isArray(types) ? types : []).forEach((typeObj) => {
    parseDbTemplatePlaceholders(typeObj?.dynamicPlaceholders).forEach((item) => {
      if (!merged.has(item.key)) {
        merged.set(item.key, item);
        return;
      }
      const current = merged.get(item.key);
      if (current && !current.description && item.description) {
        current.description = item.description;
      }
      if (current && !current.value && item.value) {
        current.value = item.value;
      }
    });
  });
  return Array.from(merged.values());
};

const escapeRegExp = (value: string) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

export const applyDbTemplatePlaceholderValues = (text: string, values: DbTemplatePlaceholderValues = {}) => {
  return Object.entries(values).reduce((next, [key, value]) => {
    const cleanKey = normalizePlaceholderKey(key);
    if (!cleanKey) return next;
    const replacement = String(value ?? '');
    const escaped = escapeRegExp(cleanKey);
    const originalKey = String(key || '').trim();
    const escapedOriginalKey = escapeRegExp(originalKey);
    const replacedByNormalizedKey = next
      .replace(new RegExp(`\\{\\{\\s*(?:(?:manual|field|snippet|parsed)\\s*:\\s*)?<?\\s*${escaped}\\s*>?\\s*\\}\\}`, 'g'), replacement)
      .replace(new RegExp(`\\$\\{\\s*(?:(?:manual|field|snippet|parsed)\\s*:\\s*)?<?\\s*${escaped}\\s*>?\\s*\\}`, 'g'), replacement)
      .replace(new RegExp(`\\{\\[\\s*<\\s*${escaped}\\s*>\\s*\\]\\}`, 'g'), replacement);
    if (!originalKey || originalKey === cleanKey) {
      return replacedByNormalizedKey;
    }
    return next
      .replace(new RegExp(escapedOriginalKey, 'g'), replacement)
      .replace(new RegExp(`\\{\\{\\s*${escaped}\\s*\\}\\}`, 'g'), replacement)
      .replace(new RegExp(`\\$\\{\\s*${escaped}\\s*\\}`, 'g'), replacement);
  }, String(text || ''));
};

export const buildDbTemplateSqlSection = (
  project: any,
  typeObj: any,
  script: any,
  placeholderValues: DbTemplatePlaceholderValues = {},
) => {
  const lines = [
    `-- 项目：${project?.projectName || project?.ID || '-'}`,
    `-- 业务类型：${typeObj?.typeName || '-'}`,
    `-- 脚本：${script?.scriptName || typeObj?.typeName || '-'}`,
    '',
    applyDbTemplatePlaceholderValues(String(script?.content || '').trim(), placeholderValues),
  ];

  const prompt = applyDbTemplatePlaceholderValues(String(typeObj?.prompt || '').trim(), placeholderValues);
  if (prompt) {
    lines.push('', '-- 提示词：', formatSqlCommentBlock(prompt));
  }

  return lines.join('\n');
};

export const buildDbTemplateSqlCopyText = (sections: string[]) => {
  return sections.map((section) => String(section || '').trim()).filter(Boolean).join('\n\n\n');
};
