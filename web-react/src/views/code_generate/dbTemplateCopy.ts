const formatSqlCommentBlock = (text: string) => {
  return String(text || '')
    .trim()
    .split(/\r?\n/)
    .map((line) => (line.trim() ? `-- ${line}` : '--'))
    .join('\n');
};

export const buildDbTemplateSqlSection = (project: any, typeObj: any, script: any) => {
  const lines = [
    `-- 项目：${project?.projectName || project?.ID || '-'}`,
    `-- 业务类型：${typeObj?.typeName || '-'}`,
    `-- 脚本：${script?.scriptName || typeObj?.typeName || '-'}`,
    '',
    String(script?.content || '').trim(),
  ];

  const prompt = String(typeObj?.prompt || '').trim();
  if (prompt) {
    lines.push('', '-- 提示词：', formatSqlCommentBlock(prompt));
  }

  return lines.join('\n');
};

export const buildDbTemplateSqlCopyText = (sections: string[]) => {
  return sections.map((section) => String(section || '').trim()).filter(Boolean).join('\n\n\n');
};
