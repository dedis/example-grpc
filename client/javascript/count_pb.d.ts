import * as jspb from "google-protobuf"

export class CountRequest extends jspb.Message {
  getValue(): number;
  setValue(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CountRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CountRequest): CountRequest.AsObject;
  static serializeBinaryToWriter(message: CountRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CountRequest;
  static deserializeBinaryFromReader(message: CountRequest, reader: jspb.BinaryReader): CountRequest;
}

export namespace CountRequest {
  export type AsObject = {
    value: number,
  }
}

export class CountResponse extends jspb.Message {
  getValue(): number;
  setValue(value: number): void;

  getCertificatesList(): Array<Uint8Array | string>;
  setCertificatesList(value: Array<Uint8Array | string>): void;
  clearCertificatesList(): void;
  addCertificates(value: Uint8Array | string, index?: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CountResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CountResponse): CountResponse.AsObject;
  static serializeBinaryToWriter(message: CountResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CountResponse;
  static deserializeBinaryFromReader(message: CountResponse, reader: jspb.BinaryReader): CountResponse;
}

export namespace CountResponse {
  export type AsObject = {
    value: number,
    certificatesList: Array<Uint8Array | string>,
  }
}

export class Counter extends jspb.Message {
  getValue(): number;
  setValue(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Counter.AsObject;
  static toObject(includeInstance: boolean, msg: Counter): Counter.AsObject;
  static serializeBinaryToWriter(message: Counter, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Counter;
  static deserializeBinaryFromReader(message: Counter, reader: jspb.BinaryReader): Counter;
}

export namespace Counter {
  export type AsObject = {
    value: number,
  }
}

export class Peer extends jspb.Message {
  getPort(): string;
  setPort(value: string): void;

  getCertificate(): Uint8Array | string;
  getCertificate_asU8(): Uint8Array;
  getCertificate_asB64(): string;
  setCertificate(value: Uint8Array | string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Peer.AsObject;
  static toObject(includeInstance: boolean, msg: Peer): Peer.AsObject;
  static serializeBinaryToWriter(message: Peer, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Peer;
  static deserializeBinaryFromReader(message: Peer, reader: jspb.BinaryReader): Peer;
}

export namespace Peer {
  export type AsObject = {
    port: string,
    certificate: Uint8Array | string,
  }
}

