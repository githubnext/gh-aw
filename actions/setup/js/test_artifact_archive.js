import { createRequire } from "module";
import zlib from "zlib";
const require = createRequire(import.meta.url);
const { crc32 } = require("./artifact_archive.cjs");

export function createTestArchive(entries) {
  const local = [];
  const central = [];
  let offset = 0;
  for (const entry of entries) {
    const name = Buffer.from(entry.name);
    const localName = Buffer.from(entry.localName || entry.name);
    const content = Buffer.from(entry.content || "");
    const data = entry.deflate ? zlib.deflateRawSync(content) : content;
    const flags = 0x800 | (entry.descriptor ? 8 : 0);
    const checksum = crc32(content);
    const header = Buffer.alloc(30);
    header.writeUInt32LE(0x04034b50);
    header.writeUInt16LE(20, 4);
    header.writeUInt16LE(flags, 6);
    header.writeUInt16LE(entry.deflate ? 8 : 0, 8);
    if (!entry.descriptor) {
      header.writeUInt32LE(checksum, 14);
      header.writeUInt32LE(data.length, 18);
      header.writeUInt32LE(content.length, 22);
    }
    header.writeUInt16LE(localName.length, 26);
    const directory = Buffer.alloc(46);
    directory.writeUInt32LE(0x02014b50);
    directory.writeUInt16LE(0x314, 4);
    directory.writeUInt16LE(20, 6);
    directory.writeUInt16LE(flags, 8);
    directory.writeUInt16LE(entry.deflate ? 8 : 0, 10);
    directory.writeUInt32LE(checksum, 16);
    directory.writeUInt32LE(data.length, 20);
    directory.writeUInt32LE(content.length, 24);
    directory.writeUInt16LE(name.length, 28);
    directory.writeUInt32LE(((entry.mode || (entry.name.endsWith("/") ? 0o40755 : 0o100644)) << 16) >>> 0, 38);
    directory.writeUInt32LE(offset, 42);
    local.push(header, localName, data);
    central.push(directory, name);
    offset += header.length + localName.length + data.length;
    if (entry.descriptor) {
      const descriptor = Buffer.alloc(16);
      descriptor.writeUInt32LE(0x08074b50);
      descriptor.writeUInt32LE(checksum, 4);
      descriptor.writeUInt32LE(data.length, 8);
      descriptor.writeUInt32LE(content.length, 12);
      local.push(descriptor);
      offset += descriptor.length;
    }
  }
  const directoryBytes = Buffer.concat(central);
  const end = Buffer.alloc(22);
  end.writeUInt32LE(0x06054b50);
  end.writeUInt16LE(entries.length, 8);
  end.writeUInt16LE(entries.length, 10);
  end.writeUInt32LE(directoryBytes.length, 12);
  end.writeUInt32LE(offset, 16);
  return Buffer.concat([...local, directoryBytes, end]);
}
