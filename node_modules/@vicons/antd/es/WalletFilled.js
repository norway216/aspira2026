import { createElementVNode as _createElementVNode, openBlock as _openBlock, createElementBlock as _createElementBlock, defineComponent } from 'vue'
const _hoisted_1 = {
  xmlns: 'http://www.w3.org/2000/svg',
  'xmlns:xlink': 'http://www.w3.org/1999/xlink',
  viewBox: '0 0 1024 1024'
}
export default defineComponent({
  name: 'WalletFilled',
  render: function render(_ctx, _cache) {
    return (
      _openBlock(),
      _createElementBlock(
        'svg',
        _hoisted_1,
        _cache[0] ||
          (_cache[0] = [
            _createElementVNode(
              'path',
              {
                d: 'M880 112H144c-17.7 0-32 14.3-32 32v736c0 17.7 14.3 32 32 32h736c17.7 0 32-14.3 32-32V144c0-17.7-14.3-32-32-32zm-32 464H528V448h320v128zm-268-64a40 40 0 1 0 80 0a40 40 0 1 0-80 0z',
                fill: 'currentColor'
              },
              null,
              -1 /* HOISTED */
            )
          ])
      )
    )
  }
})
