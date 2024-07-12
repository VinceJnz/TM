CREATE TABLE IF NOT EXISTS `at_booking` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `User_ID` int(11) NOT NULL,
  `Notes` text,
  `From_date` datetime DEFAULT NULL,
  `To_date` datetime DEFAULT NULL,
  `Booking_status_ID` int(11) NOT NULL,
  `Created` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `Modified` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;