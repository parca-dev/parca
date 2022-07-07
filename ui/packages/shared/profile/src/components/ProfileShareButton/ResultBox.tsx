import {useState} from 'react';
import cx from 'classnames';
import {Icon} from '@iconify/react';
import {Button} from '@parca/components';
import {CopyToClipboard} from 'react-copy-to-clipboard';

interface Props {
  value: string;
  className?: string;
}

let timeoutHandle: ReturnType<typeof setTimeout> | null = null;

const ResultBox = ({value, className = ''}: Props) => {
  const [isCopied, setIsCopied] = useState<boolean>(false);

  const onCopy = () => {
    setIsCopied(true);
    (window.document?.activeElement as HTMLElement)?.blur();
    if (timeoutHandle != null) {
      clearTimeout(timeoutHandle);
    }
    timeoutHandle = setTimeout(() => setIsCopied(false), 3000);
  };

  return (
    <div className={cx('flex flex-row w-full', {[className]: className?.length > 0})}>
      <span className="flex justify-center items-center border border-r-0 w-16 rounded-l">
        <Icon icon="ant-design:link-outlined" />
      </span>
      <input
        type="text"
        className="border text-sm bg-inherit w-full px-1 py-2 flex-grow"
        value={value}
        readOnly
      />
      <CopyToClipboard text={value} onCopy={onCopy}>
        <Button
          variant="link"
          className="border border-l-0 w-fit whitespace-nowrap p-4 items-center !text-indigo-600 dark:!text-indigo-400 rounded-none rounded-r"
        >
          {isCopied ? 'Copied!' : 'Copy Link'}
        </Button>
      </CopyToClipboard>
    </div>
  );
};

export default ResultBox;
