export class ToastHelper {
  static showToast({ title, description }: { title: string; description: string }) {
    const rootDiv = document.getElementById('blocks-component-root') || document.body;

    if (rootDiv) {
      const div = document.createElement('div');
      div.style.backgroundColor = '#fff';
      div.style.color = '#000';
      div.style.padding = '10px';
      div.style.border = '1px solid gainsboro';
      div.style.position = 'fixed';
      div.style.bottom = '-100px';
      div.style.maxWidth = '350px';
      div.style.boxShadow = '0 1rem 3rem rgba(0, 0, 0, .175)';
      div.style.left = '1.5rem';
      div.style.zIndex = '999999';
      div.style.transition = 'top 0.3s ease-in';
      div.style.borderRadius = '5px';

      const titleDiv = document.createElement('div');
      titleDiv.style.fontWeight = 'bold';
      titleDiv.innerText = title;
      titleDiv.style.fontSize = '14px';
      div.appendChild(titleDiv);

      const descriptionDiv = document.createElement('div');
      descriptionDiv.innerText = description;
      descriptionDiv.style.fontSize = '14px';
      div.appendChild(descriptionDiv);

      div.onclick = () => {
        try {
          rootDiv.removeChild(div);
        } catch {
          // ignore
        }
      };

      rootDiv.appendChild(div);

      setTimeout(() => {
        div.style.bottom = '1.5rem';
      }, 0);

      setTimeout(() => {
        try {
          rootDiv.removeChild(div);
        } catch {
          // ignore
        }
      }, 3_000);
    }
  }
}
