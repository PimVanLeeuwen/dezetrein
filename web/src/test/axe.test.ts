import { describe, expect, it } from 'vitest';
import { expectAccessible } from './axe';

// The helper must actually catch problems, or the page checks prove nothing.
describe('expectAccessible', () => {
  it('fails on markup with known violations', async () => {
    const box = document.createElement('div');
    box.innerHTML = '<img src="/x.png"><button></button><input type="text">';
    document.body.append(box);
    const err = await expectAccessible(box).catch((e: Error) => e);
    box.remove();
    expect(err).toBeInstanceOf(Error);
    expect(String(err)).toMatch(/image-alt/);
    expect(String(err)).toMatch(/button-name/);
    expect(String(err)).toMatch(/label/);
  });

  it('passes clean markup', async () => {
    const box = document.createElement('div');
    box.innerHTML = '<img src="/x.png" alt="Logo"><button>Ga</button><label>Naam <input type="text"></label>';
    document.body.append(box);
    await expectAccessible(box);
    box.remove();
  });
});
