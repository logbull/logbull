import { Button, Modal } from 'antd';
import type { JSX } from 'react';

interface Props {
  onConfirm(): void;
  onDecline(): void;

  description: string;
  actionButtonColor: 'blue' | 'red';

  actionText: string;
  cancelText?: string;
  hideCancelButton?: boolean;
}

export function ConfirmationComponent({
  onConfirm,
  onDecline,
  description,
  actionButtonColor,
  actionText,
  cancelText,
  hideCancelButton = false,
}: Props): JSX.Element {
  return (
    <Modal
      title="Confirmation"
      open
      onClose={() => onDecline()}
      onCancel={() => onDecline()}
      footer={<div />}
    >
      <div dangerouslySetInnerHTML={{ __html: description }} />

      <div className="mt-5 flex">
        {!hideCancelButton && (
          <Button
            className="ml-auto"
            onClick={() => onDecline()}
            danger={actionButtonColor !== 'red'}
            type="primary"
          >
            {cancelText || 'Cancel'}
          </Button>
        )}

        <Button
          className="ml-1"
          onClick={() => onConfirm()}
          danger={actionButtonColor === 'red'}
          type="primary"
        >
          {actionText}
        </Button>
      </div>
    </Modal>
  );
}
