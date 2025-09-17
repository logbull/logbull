export default class RequestOptions {
  private headers: [string, string][];
  private method: string | undefined;
  private credentials: 'include' | undefined;
  private body: string | undefined;

  constructor() {
    this.headers = [];
  }

  setMethod(method: 'GET' | 'POST' | 'DELETE' | 'PUT'): RequestOptions {
    this.method = method;
    return this;
  }

  setCredentials(credentials: 'include'): RequestOptions {
    this.credentials = credentials;
    return this;
  }

  setBody(body: string): RequestOptions {
    this.body = body;
    return this;
  }

  getBody(): string | undefined {
    return this.body;
  }

  addHeader(headerName: string, headerValue?: string): RequestOptions {
    this.headers.push([headerName, headerValue || '']);
    return this;
  }

  toRequestInit(): RequestInit {
    // Example:
    //
    // ['Autorization', 'Key']
    // ['Another-Header', 'Another-Value']
    const headersMatrix: string[][] = [];
    this.headers.forEach(([headerName, headerValue]) => {
      const headerArray: string[] = [];
      headerArray.push(headerName);
      headerArray.push(headerValue);
      headersMatrix.push(headerArray);
    });

    const requestJsonOptions: RequestInit = {
      headers: headersMatrix as [string, string][],
      cache: 'no-cache',
    };

    if (this.method) {
      requestJsonOptions.method = this.method;
    }

    if (this.credentials) {
      requestJsonOptions.credentials = this.credentials;
    }

    if (this.body) {
      requestJsonOptions.body = this.body;
    }

    return requestJsonOptions;
  }
}
