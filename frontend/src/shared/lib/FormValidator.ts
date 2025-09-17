export class FormValidator {
  static isValidEmail(email: string): boolean {
    const emailRegex = /^\S+@\S+\.\S+$/;
    return emailRegex.test(email);
  }

  static isValidUrl(url: string): boolean {
    if (url.replace('https://', '').replace('http://', '').includes('//')) {
      return false;
    }

    const urlRegex =
      /^https?:\/\/(?:www\.)?[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,9}\b(?:[-a-zA-Z0-9()@:%_+.~#?&/=]*)$/;
    return urlRegex.test(url);
  }
}
