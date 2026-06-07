import assert from 'node:assert/strict';
import { test } from 'node:test';
import { getPathGroupDeleteState, getPathGroupSwitchOptions } from './pathGroupActions.ts';

test('blocks deleting a path group when it contains path records', () => {
  const state = getPathGroupDeleteState({
    basePath: 'c12-mtp-web-service/src/main/java',
    paths: [{ ID: 1, fileUrl: 'c12-mtp-web-service/src/main/java', fileName: 'Demo.java' }],
  });

  assert.equal(state.canDelete, false);
  assert.equal(state.reason, '该子目录下还有路径数据，不能删除');
});

test('allows deleting an empty path group', () => {
  const state = getPathGroupDeleteState({
    basePath: 'c12-mtp-web-service/src/test/java',
    paths: [],
  });

  assert.equal(state.canDelete, true);
  assert.equal(state.reason, '');
});

test('builds switch options for public relative path groups', () => {
  const options = getPathGroupSwitchOptions([
    { key: 'basic-api', basePath: 'c12-mtp-basic-service/c12-mtp-basic-api/src/' },
    { key: '', basePath: '' },
    { key: 'basic-biz', basePath: 'c12-mtp-basic-service/c12-mtp-basic-biz/src/' },
  ]);

  assert.deepEqual(options, [
    { key: 'basic-api', label: 'c12-mtp-basic-service/c12-mtp-basic-api/src' },
    { key: 'basic-biz', label: 'c12-mtp-basic-service/c12-mtp-basic-biz/src' },
  ]);
});
