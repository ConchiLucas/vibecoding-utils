import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  getProjectTypeLabel,
  isBackendProject,
  matchesProjectCardSearch,
  normalizeProjectType,
  shouldShowDbTemplateActions,
} from './projectDashboardActions.ts';

test('project card search ignores legacy diskPath values', () => {
  const project = {
    projectName: '后端代码卡片',
    diskPath: '/Users/demo/legacy-output-root',
    remark: '用于代码生成',
  };

  assert.equal(matchesProjectCardSearch(project, 'legacy-output-root'), false);
  assert.equal(matchesProjectCardSearch(project, '后端'), true);
  assert.equal(matchesProjectCardSearch(project, '代码生成'), true);
});

test('normalizes legacy project cards to backend type', () => {
  assert.equal(normalizeProjectType('frontend'), 'frontend');
  assert.equal(normalizeProjectType('backend'), 'backend');
  assert.equal(normalizeProjectType(''), 'backend');
  assert.equal(normalizeProjectType(undefined), 'backend');
});

test('shows database template actions only for backend projects', () => {
  assert.equal(shouldShowDbTemplateActions({ projectType: 'backend' }), true);
  assert.equal(shouldShowDbTemplateActions({ projectType: '' }), true);
  assert.equal(shouldShowDbTemplateActions({ projectType: 'frontend' }), false);
  assert.equal(isBackendProject({ projectType: 'frontend' }), false);
});

test('returns Chinese project type labels', () => {
  assert.equal(getProjectTypeLabel('frontend'), '前端');
  assert.equal(getProjectTypeLabel('backend'), '后端');
  assert.equal(getProjectTypeLabel(undefined), '后端');
});
