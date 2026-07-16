import * as echoUtils from '../../utils';

import { PostHogBackend } from './PostHogBackend';

describe('PostHogBackend', () => {
  const postHogToken = 'phc_test';
  const postHogHost = 'https://posthog.example.com';

  let loadScriptSpy: jest.SpyInstance;

  beforeEach(() => {
    delete window.posthog;
    loadScriptSpy = jest.spyOn(echoUtils, 'loadScript').mockResolvedValue(undefined);
  });

  afterEach(() => {
    jest.restoreAllMocks();
    delete window.posthog;
  });

  it('queues initialization and events in the PostHog snippet format until the SDK loads', () => {
    new PostHogBackend({ postHogToken, postHogHost });

    window.posthog?.identify?.('user-1');
    window.posthog?.capture?.('test_interaction', { source: 'test' });

    const posthog = window.posthog ?? {};
    expect(loadScriptSpy).toHaveBeenCalledWith(`${postHogHost}/static/array.js`, true);
    expect(Reflect.get(posthog, '_i')).toEqual([[postHogToken, { api_host: postHogHost }, 'posthog']]);
    expect(Reflect.get(posthog, 'length')).toBe(2);
    expect(Reflect.get(posthog, '0')).toEqual(['identify', 'user-1']);
    expect(Reflect.get(posthog, '1')).toEqual(['capture', 'test_interaction', { source: 'test' }]);
  });
});
