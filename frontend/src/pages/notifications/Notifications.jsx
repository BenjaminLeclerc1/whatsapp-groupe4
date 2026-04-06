import React from 'react';
import { useApp } from '.../../context/AppContext';

const NotificationBadge = ({ chatId }) => {
  const { notifications } = useApp();

  // On compte combien de notifications concernent ce chat précis
  const chatNotifications = notifications.filter(n => n.chat_id === chatId);
  
  if (chatNotifications.length === 0) return null;

  return (
    <div className="notification-badge">
      {chatNotifications.length}
    </div>
  );
};

export default NotificationBadge;