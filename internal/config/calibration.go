package config

import "wim-service/internal/vision"

// GetCameraCalibration returns a CameraCalibration object from config
func (c *Config) GetCameraCalibration() *vision.CameraCalibration {
	calibration := vision.NewCameraCalibration()

	calibration.LoadFromConfig(
		c.CameraFocalLength,
		c.CameraImageWidth,
		c.CameraImageHeight,
		c.CameraHeight,
		c.CameraTiltAngle,
		c.CameraRefPixelLength,
		c.CameraRefRealLength,
		c.CameraRefDistance,
	)

	return calibration
}
