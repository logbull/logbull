import GitHubButton from 'react-github-btn';

export function AuthNavbarComponent() {
  return (
    <div className="m-3 flex h-[65px] items-center p-3">
      <div className="flex items-center gap-3 hover:opacity-80">
        <a href="https://logbull.com" target="_blank" rel="noreferrer">
          <img className="h-[35px] w-[35px]" src="/logo.svg" />
        </a>

        <div className="text-xl font-bold">
          <a href="https://logbull.com" className="!text-black" target="_blank" rel="noreferrer">
            Log Bull
          </a>
        </div>
      </div>

      <div className="mr-3 ml-auto flex items-center gap-5">
        <a
          className="!text-black hover:opacity-80"
          href="https://t.me/logbull_community"
          target="_blank"
          rel="noreferrer"
        >
          Community
        </a>

        <div className="mt-1">
          <GitHubButton
            href="https://github.com/logbull/logbull"
            data-icon="octicon-star"
            data-size="large"
            data-show-count="true"
            aria-label="Star Log Bull on GitHub"
          >
            &nbsp;Star on GitHub
          </GitHubButton>
        </div>
      </div>
    </div>
  );
}
