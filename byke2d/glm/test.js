// Assume add.wasm file exists that contains a single function adding 2 provided arguments
import fs from 'node:fs/promises';

// Use readFile to read contents of the "add.wasm" file
const wasmBuffer = await fs.readFile('mat4f.wasm');

const memory = new WebAssembly.Memory({
    initial: 65536,
    maximum: 65536,
});

// Use the WebAssembly.instantiate method to instantiate the WebAssembly module
const wasmModule = await WebAssembly.instantiate(wasmBuffer, {
    env: {memory},
});

const data = new DataView(memory.buffer);

const m = 64;
const out = 512;

const at = (base, col, row) => base + 4 * (4 * col + row);

function print(base) {
    let str = "";
    for (let r = 0; r < 4; r++) {
        for (let c = 0; c < 4; c++) {
            str += data.getFloat32(at(base, c, r), true) + " ";
        }

        str += "\n";
    }

    return console.log(str);
}

data.setFloat32(at(m, 0, 0), 1, true);
data.setFloat32(at(m, 1, 1), 2, true);
data.setFloat32(at(m, 2, 2), 3, true);
data.setFloat32(at(m, 3, 3), 1, true);
print(m)

const {mat4f_mul} = wasmModule.instance.exports;
mat4f_mul(m, m, out);
print(out);
