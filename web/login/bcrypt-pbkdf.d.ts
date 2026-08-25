declare module "bcrypt-pbkdf" {
  const bpf: {
    pbkdf: (
      pass: Uint8Array,
      passlen: number,
      salt: Uint8Array,
      saltlen: number,
      key: Uint8Array,
      keylen: number,
      rounds: number,
    ) => number;
  };
  export default bpf;
}
